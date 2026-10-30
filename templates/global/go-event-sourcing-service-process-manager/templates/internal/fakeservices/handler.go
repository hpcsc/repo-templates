package fakeservices

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// The input of a payment selects the path that the fakes take.
const (
	fraudLimit         = 10000
	sanctionedSuffix   = "0000"
	unscreenedSuffix   = "1111"
	returnedSuffix     = "2222"
	unsettledSuffix    = "3333"
	unansweredSuffix   = "4444"
	webhookAttempts    = 3
	webhookAttemptWait = 500 * time.Millisecond
)

type PaymentRequest struct {
	PaymentID    string `json:"paymentId"`
	Amount       int64  `json:"amount"`
	CreditorIBAN string `json:"creditorIban"`
	WebhookURL   string `json:"webhookUrl"`
}

type CheckResult struct {
	PaymentID string `json:"paymentId"`
	Passed    bool   `json:"passed"`
	Reason    string `json:"reason,omitempty"`
}

type NetworkStatus struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

type SubmitAnswer struct {
	Reference string `json:"reference"`
}

func NewHandler(token string, delay time.Duration, logger *slog.Logger) http.Handler {
	f := &fakes{
		token:    token,
		delay:    delay,
		client:   &http.Client{Timeout: 5 * time.Second},
		logger:   logger,
		answers:  make(map[string]any),
		statuses: make(map[string]NetworkStatus),
	}

	r := chi.NewRouter()
	r.Post("/fraud/checks", f.fraudCheck)
	r.Post("/sanctions/screenings", f.sanctionsScreening)
	r.Post("/network/payments", f.networkPayment)
	r.Get("/network/payments/{reference}", f.networkStatus)
	return r
}

type fakes struct {
	token  string
	delay  time.Duration
	client *http.Client
	logger *slog.Logger

	mutex    sync.Mutex
	answers  map[string]any
	statuses map[string]NetworkStatus
}

func (f *fakes) fraudCheck(w http.ResponseWriter, r *http.Request) {
	f.answerOnce(w, r, func(p PaymentRequest, key string) any {
		result := CheckResult{PaymentID: p.PaymentID, Passed: true}
		if p.Amount > fraudLimit {
			result = CheckResult{PaymentID: p.PaymentID, Passed: false, Reason: "amount above the limit of the fraud check"}
		}
		f.callLater(p.WebhookURL, key, result)
		return struct{}{}
	})
}

func (f *fakes) sanctionsScreening(w http.ResponseWriter, r *http.Request) {
	f.answerOnce(w, r, func(p PaymentRequest, key string) any {
		switch {
		case strings.HasSuffix(p.CreditorIBAN, unscreenedSuffix):
			// The screening never answers, so the checks of the payment wait.
		case strings.HasSuffix(p.CreditorIBAN, sanctionedSuffix):
			f.callLater(p.WebhookURL, key, CheckResult{PaymentID: p.PaymentID, Passed: false, Reason: "creditor on the sanctions list"})
		default:
			f.callLater(p.WebhookURL, key, CheckResult{PaymentID: p.PaymentID, Passed: true})
		}
		return struct{}{}
	})
}

func (f *fakes) networkPayment(w http.ResponseWriter, r *http.Request) {
	f.answerOnce(w, r, func(p PaymentRequest, key string) any {
		reference := "N-" + strings.ToUpper(uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String()[:8])
		switch {
		case strings.HasSuffix(p.CreditorIBAN, unsettledSuffix):
			f.statuses[reference] = NetworkStatus{PaymentID: p.PaymentID, Status: "settled"}
		case strings.HasSuffix(p.CreditorIBAN, unansweredSuffix):
			f.statuses[reference] = NetworkStatus{PaymentID: p.PaymentID, Status: "pending"}
		case strings.HasSuffix(p.CreditorIBAN, returnedSuffix):
			f.statuses[reference] = NetworkStatus{PaymentID: p.PaymentID, Status: "returned", Reason: "account closed at the other bank"}
			f.callLater(p.WebhookURL, key, f.statuses[reference])
		default:
			f.statuses[reference] = NetworkStatus{PaymentID: p.PaymentID, Status: "settled"}
			f.callLater(p.WebhookURL, key, f.statuses[reference])
		}
		return SubmitAnswer{Reference: reference}
	})
}

func (f *fakes) networkStatus(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	status, ok := f.statuses[chi.URLParam(r, "reference")]
	f.mutex.Unlock()
	if !ok {
		http.Error(w, "no payment has the reference", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// answerOnce decides once for each Idempotency-Key, and gives a repeated request the same answer
// and no second webhook. It holds the mutex while decide runs.
func (f *fakes) answerOnce(w http.ResponseWriter, r *http.Request, decide func(p PaymentRequest, key string) any) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		http.Error(w, "the Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	var p PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "received invalid request body", http.StatusBadRequest)
		return
	}

	f.mutex.Lock()
	answer, ok := f.answers[r.URL.Path+" "+key]
	if !ok {
		answer = decide(p, key)
		f.answers[r.URL.Path+" "+key] = answer
	}
	f.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(answer)
}

func (f *fakes) callLater(url string, key string, body any) {
	time.AfterFunc(f.delay, func() {
		if err := f.call(url, key, body); err != nil {
			f.logger.Error(fmt.Sprintf("failed to call the webhook %s: %v", url, err))
		}
	})
}

func (f *fakes) call(url string, key string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= webhookAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(webhookAttemptWait)
		}
		if lastErr = f.post(url, key, payload); lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func (f *fakes) post(url string, key string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Idempotency-Key", key)

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("the webhook answered %d", resp.StatusCode)
	}
	return nil
}
