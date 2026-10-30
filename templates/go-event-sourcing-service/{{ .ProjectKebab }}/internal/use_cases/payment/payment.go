package payment

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
)

var (
	ErrAlreadyRequested = errors.New("payment is already requested")
	ErrNotRequested     = errors.New("payment is not requested")
	ErrNotSubmitted     = errors.New("payment is not submitted")
	ErrInvalidAmount    = errors.New("amount must be more than 0")
	ErrNoCreditor       = errors.New("creditor IBAN must not be empty")
	ErrUnknownAccount   = errors.New("account does not exist")
)

var _ domain.Aggregate = (*Payment)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("payment-%s", id)
}

type Timeouts struct {
	Checks     time.Duration
	Submission time.Duration
	Settlement time.Duration
}

// MaxAttempts is the number of times that a payment asks for one stage before it fails or escalates.
const MaxAttempts = 3

var DefaultTimeouts = Timeouts{
	Checks:     10 * time.Second,
	Submission: 10 * time.Second,
	Settlement: 10 * time.Second,
}

type stage int

const (
	requested stage = iota + 1
	funded
	cleared
	submitted
	escalated
	settled
	returned
	refused
	failed
)

type Payment struct {
	domain.AggregateBase

	id              uuid.UUID
	accountID       uuid.UUID
	stage           stage
	fraudPassed     bool
	screeningPassed bool
	attempt         int
}

func (p *Payment) ID() uuid.UUID {
	return p.id
}

func (p *Payment) Request(id uuid.UUID, accountID uuid.UUID, creditorIBAN string, amount int64) error {
	if p.id != uuid.Nil {
		return ErrAlreadyRequested
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if creditorIBAN == "" {
		return ErrNoCreditor
	}

	p.record(domain.NewEvent(&event.PaymentRequested{
		PaymentID:    id,
		AccountID:    accountID,
		CreditorIBAN: creditorIBAN,
		Amount:       amount,
	}))
	return nil
}

func (p *Payment) RecordFunding(held bool, reason string, checksDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != requested {
		return nil
	}

	if !held {
		p.record(domain.NewEvent(&event.PaymentFailed{PaymentID: p.id, AccountID: p.accountID, Reason: reason}))
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentFunded{PaymentID: p.id, ChecksDeadline: checksDeadline}))
	return nil
}

func (p *Payment) RecordFraudResult(passed bool, reason string, submissionDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != funded || p.fraudPassed {
		return nil
	}

	if !passed {
		p.record(domain.NewEvent(&event.FraudCheckFailed{PaymentID: p.id, Reason: reason}))
		p.refuse(reason)
		return nil
	}

	p.record(domain.NewEvent(&event.FraudCheckPassed{PaymentID: p.id}))
	p.clearAfterChecks(submissionDeadline)
	return nil
}

func (p *Payment) RecordScreeningResult(passed bool, reason string, submissionDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != funded || p.screeningPassed {
		return nil
	}

	if !passed {
		p.record(domain.NewEvent(&event.SanctionsScreeningFailed{PaymentID: p.id, Reason: reason}))
		p.refuse(reason)
		return nil
	}

	p.record(domain.NewEvent(&event.SanctionsScreeningPassed{PaymentID: p.id}))
	p.clearAfterChecks(submissionDeadline)
	return nil
}

func (p *Payment) RecordSubmission(networkReference string, settlementDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != cleared {
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentSubmitted{
		PaymentID:          p.id,
		NetworkReference:   networkReference,
		SettlementDeadline: settlementDeadline,
	}))
	return nil
}

func (p *Payment) RecordSettlement() error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if !p.awaitsNetwork() {
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentSettled{PaymentID: p.id, AccountID: p.accountID}))
	return nil
}

func (p *Payment) RecordReturn(reason string) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if !p.awaitsNetwork() {
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentReturned{PaymentID: p.id, AccountID: p.accountID, Reason: reason}))
	return nil
}

func (p *Payment) TimeOutChecks(attempt int, checksDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != funded || attempt != p.attempt {
		return nil
	}

	if attempt >= MaxAttempts {
		p.record(domain.NewEvent(&event.PaymentFailed{PaymentID: p.id, AccountID: p.accountID, Reason: "the checks did not answer"}))
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentChecksRetried{PaymentID: p.id, Attempt: attempt + 1, ChecksDeadline: checksDeadline}))
	return nil
}

// TimeOutSubmission escalates and never fails the payment, because the network can have it already.
func (p *Payment) TimeOutSubmission(attempt int, submissionDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != cleared || attempt != p.attempt {
		return nil
	}

	if attempt >= MaxAttempts {
		p.record(domain.NewEvent(&event.PaymentEscalated{PaymentID: p.id, Reason: "the payment network did not take the payment"}))
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentSubmissionRetried{PaymentID: p.id, Attempt: attempt + 1, SubmissionDeadline: submissionDeadline}))
	return nil
}

func (p *Payment) TimeOutSettlement(attempt int, settlementDeadline time.Time) error {
	if p.id == uuid.Nil {
		return ErrNotRequested
	}
	if p.stage != submitted || attempt != p.attempt {
		return nil
	}

	if attempt >= MaxAttempts {
		p.record(domain.NewEvent(&event.PaymentEscalated{PaymentID: p.id, Reason: "the payment network did not settle the payment"}))
		return nil
	}

	p.record(domain.NewEvent(&event.PaymentStatusQueried{PaymentID: p.id, Attempt: attempt + 1, SettlementDeadline: settlementDeadline}))
	return nil
}

func (p *Payment) Apply(e domain.Event) {
	switch payload := e.Payload.(type) {
	case *event.PaymentRequested:
		p.id = payload.PaymentID
		p.accountID = payload.AccountID
		p.stage = requested
	case *event.PaymentFunded:
		p.stage = funded
		p.attempt = 1
	case *event.PaymentChecksRetried:
		p.attempt = payload.Attempt
	case *event.PaymentFailed:
		p.stage = failed
	case *event.FraudCheckPassed:
		p.fraudPassed = true
	case *event.SanctionsScreeningPassed:
		p.screeningPassed = true
	case *event.PaymentCleared:
		p.stage = cleared
		p.attempt = 1
	case *event.PaymentSubmissionRetried:
		p.attempt = payload.Attempt
	case *event.PaymentRefused:
		p.stage = refused
	case *event.PaymentSubmitted:
		p.stage = submitted
		p.attempt = 1
	case *event.PaymentStatusQueried:
		p.attempt = payload.Attempt
	case *event.PaymentEscalated:
		p.stage = escalated
	case *event.PaymentSettled:
		p.stage = settled
	case *event.PaymentReturned:
		p.stage = returned
	}
}

func (p *Payment) clearAfterChecks(submissionDeadline time.Time) {
	if p.fraudPassed && p.screeningPassed {
		p.record(domain.NewEvent(&event.PaymentCleared{PaymentID: p.id, SubmissionDeadline: submissionDeadline}))
	}
}

func (p *Payment) refuse(reason string) {
	p.record(domain.NewEvent(&event.PaymentRefused{PaymentID: p.id, AccountID: p.accountID, Reason: reason}))
}

// A payment waits for the network from the time it clears, because the network can settle it
// before the submission gateway records its reference. An escalated payment still waits, because
// the network can settle it late.
func (p *Payment) awaitsNetwork() bool {
	return p.stage == cleared || p.stage == submitted || p.stage == escalated
}

func (p *Payment) record(e domain.Event) {
	p.Apply(e)
	p.Record(e)
}
