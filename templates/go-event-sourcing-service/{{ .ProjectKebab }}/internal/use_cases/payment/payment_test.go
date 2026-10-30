//go:build unit

package payment_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/event"
	"github.com/stretchr/testify/require"
)

var (
	paymentID = uuid.MustParse("5a7e2c91-3d4b-4f86-b1e0-6c9d8a2f4e17")
	accountID = uuid.MustParse("9d3b6f20-1e8c-4a57-8f4d-2b7a5c0e9d38")
	deadline  = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
)

func TestPayment(t *testing.T) {
	t.Run("request", func(t *testing.T) {
		t.Run("records the request", func(t *testing.T) {
			var p payment.Payment

			require.NoError(t, p.Request(paymentID, accountID, "DE89370400440532013000", 300))

			require.Equal(t, []any{&event.PaymentRequested{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300}}, recorded(&p))
		})

		t.Run("rejects a second request of the same payment", func(t *testing.T) {
			p := given(requested())

			require.ErrorIs(t, p.Request(paymentID, accountID, "DE89370400440532013000", 300), payment.ErrAlreadyRequested)
		})

		t.Run("rejects an amount that is not more than 0", func(t *testing.T) {
			var p payment.Payment

			require.ErrorIs(t, p.Request(paymentID, accountID, "DE89370400440532013000", 0), payment.ErrInvalidAmount)
		})

		t.Run("rejects an empty creditor IBAN", func(t *testing.T) {
			var p payment.Payment

			require.ErrorIs(t, p.Request(paymentID, accountID, "", 300), payment.ErrNoCreditor)
		})
	})

	t.Run("record funding", func(t *testing.T) {
		t.Run("funds the payment and sets the deadline of the checks when the account holds the amount", func(t *testing.T) {
			p := given(requested())

			require.NoError(t, p.RecordFunding(true, "", deadline))

			require.Equal(t, []any{&event.PaymentFunded{PaymentID: paymentID, ChecksDeadline: deadline}}, recorded(p))
		})

		t.Run("fails the payment with the reason of the account when the account rejects the hold", func(t *testing.T) {
			p := given(requested())

			require.NoError(t, p.RecordFunding(false, "balance is less than the amount", deadline))

			require.Equal(t, []any{&event.PaymentFailed{PaymentID: paymentID, AccountID: accountID, Reason: "balance is less than the amount"}}, recorded(p))
		})

		t.Run("records nothing when the funding arrives again", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordFunding(true, "", deadline))

			require.Empty(t, recorded(p))
		})

		t.Run("rejects the funding of a payment that is not requested", func(t *testing.T) {
			var p payment.Payment

			require.ErrorIs(t, p.RecordFunding(true, "", deadline), payment.ErrNotRequested)
		})
	})

	t.Run("record fraud result", func(t *testing.T) {
		t.Run("records the pass and waits for the screening", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordFraudResult(true, "", deadline))

			require.Equal(t, []any{&event.FraudCheckPassed{PaymentID: paymentID}}, recorded(p))
		})

		t.Run("clears the payment and sets the deadline of the submission when the screening already passed", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.SanctionsScreeningPassed{PaymentID: paymentID}))

			require.NoError(t, p.RecordFraudResult(true, "", deadline))

			require.Equal(t, []any{
				&event.FraudCheckPassed{PaymentID: paymentID},
				&event.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: deadline},
			}, recorded(p))
		})

		t.Run("refuses the payment when the check fails", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordFraudResult(false, "amount above the limit", deadline))

			require.Equal(t, []any{
				&event.FraudCheckFailed{PaymentID: paymentID, Reason: "amount above the limit"},
				&event.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "amount above the limit"},
			}, recorded(p))
		})

		t.Run("records nothing when the result arrives again", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.FraudCheckPassed{PaymentID: paymentID}))

			require.NoError(t, p.RecordFraudResult(true, "", deadline))

			require.Empty(t, recorded(p))
		})

		t.Run("records nothing when the result arrives after the payment ended", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "name on the sanctions list"}))

			require.NoError(t, p.RecordFraudResult(true, "", deadline))

			require.Empty(t, recorded(p))
		})

		t.Run("records nothing when the result arrives before the funding", func(t *testing.T) {
			p := given(requested())

			require.NoError(t, p.RecordFraudResult(true, "", deadline))

			require.Empty(t, recorded(p))
		})
	})

	t.Run("record screening result", func(t *testing.T) {
		t.Run("records the pass and waits for the fraud check", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordScreeningResult(true, "", deadline))

			require.Equal(t, []any{&event.SanctionsScreeningPassed{PaymentID: paymentID}}, recorded(p))
		})

		t.Run("clears the payment when the fraud check already passed", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.FraudCheckPassed{PaymentID: paymentID}))

			require.NoError(t, p.RecordScreeningResult(true, "", deadline))

			require.Equal(t, []any{
				&event.SanctionsScreeningPassed{PaymentID: paymentID},
				&event.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: deadline},
			}, recorded(p))
		})

		t.Run("refuses the payment when the screening fails, also after the fraud check passed", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.FraudCheckPassed{PaymentID: paymentID}))

			require.NoError(t, p.RecordScreeningResult(false, "name on the sanctions list", deadline))

			require.Equal(t, []any{
				&event.SanctionsScreeningFailed{PaymentID: paymentID, Reason: "name on the sanctions list"},
				&event.PaymentRefused{PaymentID: paymentID, AccountID: accountID, Reason: "name on the sanctions list"},
			}, recorded(p))
		})

		t.Run("records nothing when the result arrives again", func(t *testing.T) {
			p := given(requested(), funded(), domain.NewEvent(&event.SanctionsScreeningPassed{PaymentID: paymentID}))

			require.NoError(t, p.RecordScreeningResult(false, "name on the sanctions list", deadline))

			require.Empty(t, recorded(p))
		})
	})

	t.Run("record submission", func(t *testing.T) {
		t.Run("records the reference of the network and sets the deadline of the settlement", func(t *testing.T) {
			p := given(requested(), funded(), cleared())

			require.NoError(t, p.RecordSubmission("N-8812", deadline))

			require.Equal(t, []any{&event.PaymentSubmitted{PaymentID: paymentID, NetworkReference: "N-8812", SettlementDeadline: deadline}}, recorded(p))
		})

		t.Run("records nothing when the submission arrives again", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted())

			require.NoError(t, p.RecordSubmission("N-8812", deadline))

			require.Empty(t, recorded(p))
		})

		t.Run("records nothing when the payment is not cleared", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordSubmission("N-8812", deadline))

			require.Empty(t, recorded(p))
		})
	})

	t.Run("record settlement", func(t *testing.T) {
		t.Run("settles a submitted payment", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted())

			require.NoError(t, p.RecordSettlement())

			require.Equal(t, []any{&event.PaymentSettled{PaymentID: paymentID, AccountID: accountID}}, recorded(p))
		})

		t.Run("settles a cleared payment when the settlement arrives before the submission", func(t *testing.T) {
			p := given(requested(), funded(), cleared())

			require.NoError(t, p.RecordSettlement())

			require.Equal(t, []any{&event.PaymentSettled{PaymentID: paymentID, AccountID: accountID}}, recorded(p))
		})

		t.Run("records nothing when the payment already settled", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), domain.NewEvent(&event.PaymentSettled{PaymentID: paymentID, AccountID: accountID}))

			require.NoError(t, p.RecordSettlement())

			require.Empty(t, recorded(p))
		})

		t.Run("records nothing when the payment is not cleared", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.RecordSettlement())

			require.Empty(t, recorded(p))
		})
	})

	t.Run("record return", func(t *testing.T) {
		t.Run("records the return with the reason of the other bank", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted())

			require.NoError(t, p.RecordReturn("account closed at the other bank"))

			require.Equal(t, []any{&event.PaymentReturned{PaymentID: paymentID, AccountID: accountID, Reason: "account closed at the other bank"}}, recorded(p))
		})

		t.Run("records nothing when the payment already settled", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), domain.NewEvent(&event.PaymentSettled{PaymentID: paymentID, AccountID: accountID}))

			require.NoError(t, p.RecordReturn("account closed at the other bank"))

			require.Empty(t, recorded(p))
		})
	})
	t.Run("time out checks", func(t *testing.T) {
		t.Run("asks for the checks again with a new deadline", func(t *testing.T) {
			p := given(requested(), funded())

			require.NoError(t, p.TimeOutChecks(1, deadline))

			require.Equal(t, []any{&event.PaymentChecksRetried{PaymentID: paymentID, Attempt: 2, ChecksDeadline: deadline}}, recorded(p))
		})

		t.Run("fails the payment after the last attempt", func(t *testing.T) {
			p := given(requested(), funded(), checksRetried(2), checksRetried(3))

			require.NoError(t, p.TimeOutChecks(3, deadline))

			require.Equal(t, []any{&event.PaymentFailed{PaymentID: paymentID, AccountID: accountID, Reason: "the checks did not answer"}}, recorded(p))
		})

		t.Run("records nothing for an attempt that is not the last one asked", func(t *testing.T) {
			p := given(requested(), funded(), checksRetried(2))

			require.NoError(t, p.TimeOutChecks(1, deadline))

			require.Empty(t, recorded(p))
		})

		t.Run("records nothing after the payment cleared", func(t *testing.T) {
			p := given(requested(), funded(), cleared())

			require.NoError(t, p.TimeOutChecks(1, deadline))

			require.Empty(t, recorded(p))
		})
	})

	t.Run("time out submission", func(t *testing.T) {
		t.Run("asks to send the payment again with a new deadline", func(t *testing.T) {
			p := given(requested(), funded(), cleared())

			require.NoError(t, p.TimeOutSubmission(1, deadline))

			require.Equal(t, []any{&event.PaymentSubmissionRetried{PaymentID: paymentID, Attempt: 2, SubmissionDeadline: deadline}}, recorded(p))
		})

		t.Run("escalates the payment after the last attempt, and does not fail it", func(t *testing.T) {
			p := given(requested(), funded(), cleared(),
				domain.NewEvent(&event.PaymentSubmissionRetried{PaymentID: paymentID, Attempt: 2, SubmissionDeadline: deadline}),
				domain.NewEvent(&event.PaymentSubmissionRetried{PaymentID: paymentID, Attempt: 3, SubmissionDeadline: deadline}))

			require.NoError(t, p.TimeOutSubmission(3, deadline))

			require.Equal(t, []any{&event.PaymentEscalated{PaymentID: paymentID, Reason: "the payment network did not take the payment"}}, recorded(p))
		})
	})

	t.Run("time out settlement", func(t *testing.T) {
		t.Run("asks the network for the status with a new deadline", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted())

			require.NoError(t, p.TimeOutSettlement(1, deadline))

			require.Equal(t, []any{&event.PaymentStatusQueried{PaymentID: paymentID, Attempt: 2, SettlementDeadline: deadline}}, recorded(p))
		})

		t.Run("escalates the payment after the last attempt", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), statusQueried(2), statusQueried(3))

			require.NoError(t, p.TimeOutSettlement(3, deadline))

			require.Equal(t, []any{&event.PaymentEscalated{PaymentID: paymentID, Reason: "the payment network did not settle the payment"}}, recorded(p))
		})
	})

	t.Run("escalated payment", func(t *testing.T) {
		t.Run("settles when the settlement arrives after the escalation", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), escalated())

			require.NoError(t, p.RecordSettlement())

			require.Equal(t, []any{&event.PaymentSettled{PaymentID: paymentID, AccountID: accountID}}, recorded(p))
		})

		t.Run("records the return when the return arrives after the escalation", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), escalated())

			require.NoError(t, p.RecordReturn("account closed at the other bank"))

			require.Equal(t, []any{&event.PaymentReturned{PaymentID: paymentID, AccountID: accountID, Reason: "account closed at the other bank"}}, recorded(p))
		})

		t.Run("records nothing for a later timeout", func(t *testing.T) {
			p := given(requested(), funded(), cleared(), submitted(), statusQueried(2), statusQueried(3), escalated())

			require.NoError(t, p.TimeOutSettlement(3, deadline))

			require.Empty(t, recorded(p))
		})
	})
}

func given(history ...domain.Event) *payment.Payment {
	var p payment.Payment
	for _, e := range history {
		p.Apply(e)
	}
	p.MarkCommitted(uint64(len(history)))
	return &p
}

func requested() domain.Event {
	return domain.NewEvent(&event.PaymentRequested{PaymentID: paymentID, AccountID: accountID, CreditorIBAN: "DE89370400440532013000", Amount: 300})
}

func funded() domain.Event {
	return domain.NewEvent(&event.PaymentFunded{PaymentID: paymentID, ChecksDeadline: deadline})
}

func cleared() domain.Event {
	return domain.NewEvent(&event.PaymentCleared{PaymentID: paymentID, SubmissionDeadline: deadline})
}

func submitted() domain.Event {
	return domain.NewEvent(&event.PaymentSubmitted{PaymentID: paymentID, NetworkReference: "N-8812", SettlementDeadline: deadline})
}

func recorded(p *payment.Payment) []any {
	payloads := []any{}
	for _, e := range p.Recorded() {
		payloads = append(payloads, e.Payload)
	}
	return payloads
}

func checksRetried(attempt int) domain.Event {
	return domain.NewEvent(&event.PaymentChecksRetried{PaymentID: paymentID, Attempt: attempt, ChecksDeadline: deadline})
}

func statusQueried(attempt int) domain.Event {
	return domain.NewEvent(&event.PaymentStatusQueried{PaymentID: paymentID, Attempt: attempt, SettlementDeadline: deadline})
}

func escalated() domain.Event {
	return domain.NewEvent(&event.PaymentEscalated{PaymentID: paymentID, Reason: "the payment network did not settle the payment"})
}
