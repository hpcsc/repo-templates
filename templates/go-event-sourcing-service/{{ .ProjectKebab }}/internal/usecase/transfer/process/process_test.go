//go:build unit

package process_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
	"github.com/stretchr/testify/require"
)

var (
	transferID = uuid.MustParse("0b8e3d1a-5c47-4e2f-8a6b-9d1c2e3f4a50")
	from       = uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71")
	to         = uuid.MustParse("c3d4e5f6-1a2b-4c3d-8e9f-0a1b2c3d4e5f")
)

func TestProcess(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("asks the source account for the debit", func(t *testing.T) {
			p, requests := process.Start(&transferEvent.TransferStarted{TransferID: transferID, From: from, To: to, Amount: 300}, 1)

			require.Equal(t, process.Debiting, p.State())
			require.Equal(t, []any{&transferEvent.TransferDebitRequested{TransferID: transferID, AccountID: from, Amount: 300}}, payloads(requests))
		})
	})

	t.Run("advance", func(t *testing.T) {
		t.Run("asks the target account for the credit after the debit", func(t *testing.T) {
			p := started()

			requests := p.Advance(debited())

			require.Equal(t, process.Crediting, p.State())
			require.Equal(t, []any{&transferEvent.TransferCreditRequested{TransferID: transferID, AccountID: to, Amount: 300}}, payloads(requests))
		})

		t.Run("completes the transfer after the credit", func(t *testing.T) {
			p := started()
			p.Advance(debited())

			requests := p.Advance(domain.NewEvent(&accountEvent.TransferCredited{AccountID: to, TransferID: transferID, Amount: 300}))

			require.Equal(t, process.Completed, p.State())
			require.Equal(t, []any{&transferEvent.TransferCompleted{TransferID: transferID}}, payloads(requests))
		})

		t.Run("fails with nothing to undo when the debit is rejected", func(t *testing.T) {
			p := started()

			requests := p.Advance(domain.NewEvent(&accountEvent.TransferDebitRejected{AccountID: from, TransferID: transferID, Amount: 300, Reason: "balance is less than the amount"}))

			require.Equal(t, process.Failed, p.State())
			require.Equal(t, []any{&transferEvent.TransferFailed{TransferID: transferID, Reason: "balance is less than the amount"}}, payloads(requests))
		})

		t.Run("asks the source account for a refund when the credit is rejected", func(t *testing.T) {
			p := started()
			p.Advance(debited())

			requests := p.Advance(creditRejected("account is closed"))

			require.Equal(t, process.Refunding, p.State())
			require.Equal(t, []any{&transferEvent.TransferRefundRequested{TransferID: transferID, AccountID: from, Amount: 300}}, payloads(requests))
		})

		t.Run("fails with the reason of the rejected credit after the refund", func(t *testing.T) {
			p := started()
			p.Advance(debited())
			p.Advance(creditRejected("account is closed"))

			requests := p.Advance(domain.NewEvent(&accountEvent.TransferRefunded{AccountID: from, TransferID: transferID, Amount: 300}))

			require.Equal(t, process.Failed, p.State())
			require.Equal(t, "account is closed", p.Reason())
			require.Equal(t, []any{&transferEvent.TransferFailed{TransferID: transferID, Reason: "account is closed"}}, payloads(requests))
		})

		t.Run("asks for nothing when an event arrives again", func(t *testing.T) {
			p := started()
			p.Advance(debited())

			requests := p.Advance(debited())

			require.Empty(t, requests)
			require.Equal(t, process.Crediting, p.State())
		})
	})
}

func started() *process.Process {
	p, _ := process.Start(&transferEvent.TransferStarted{TransferID: transferID, From: from, To: to, Amount: 300}, 1)
	return p
}

func debited() domain.Event {
	return domain.NewEvent(&accountEvent.TransferDebited{AccountID: from, TransferID: transferID, Amount: 300})
}

func creditRejected(reason string) domain.Event {
	return domain.NewEvent(&accountEvent.TransferCreditRejected{AccountID: to, TransferID: transferID, Amount: 300, Reason: reason})
}

func payloads(events []domain.Event) []any {
	result := make([]any, 0, len(events))
	for _, e := range events {
		result = append(result, e.Payload)
	}
	return result
}
