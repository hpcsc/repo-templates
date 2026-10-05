//go:build unit

package process_test

import (
	"context"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction/dispatch"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
	"github.com/stretchr/testify/require"
)

func TestDispatcher(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("sends the requested debit with a command ID made from the request", func(t *testing.T) {
			handler := fake.NewHandler[accountCommand.DebitForTransfer]()
			bus := command.NewBus()
			bus.Register[accountCommand.DebitForTransfer](handler)
			d := process.NewDispatcher(bus)
			requested := &domain.Event{ID: "request-1", Payload: &transferEvent.TransferDebitRequested{TransferID: transferID, AccountID: from, Amount: 300}}

			require.NoError(t, d.Handle(context.Background(), requested))

			cmd := handler.TriggeredWithCommand()
			require.Equal(t, accountCommand.DebitForTransfer{AccountID: from, TransferID: transferID, Amount: 300}, cmd.Payload)
			require.Equal(t, dispatch.CommandIDFor(d.Name(), requested), cmd.ID)
		})

		t.Run("sends the requested credit and the requested refund", func(t *testing.T) {
			credit := fake.NewHandler[accountCommand.CreditForTransfer]()
			refund := fake.NewHandler[accountCommand.RefundTransfer]()
			bus := command.NewBus()
			bus.Register[accountCommand.CreditForTransfer](credit)
			bus.Register[accountCommand.RefundTransfer](refund)
			d := process.NewDispatcher(bus)

			require.NoError(t, d.Handle(context.Background(), &domain.Event{ID: "request-2", Payload: &transferEvent.TransferCreditRequested{TransferID: transferID, AccountID: to, Amount: 300}}))
			require.NoError(t, d.Handle(context.Background(), &domain.Event{ID: "request-3", Payload: &transferEvent.TransferRefundRequested{TransferID: transferID, AccountID: from, Amount: 300}}))

			require.Equal(t, accountCommand.CreditForTransfer{AccountID: to, TransferID: transferID, Amount: 300}, credit.TriggeredWithCommand().Payload)
			require.Equal(t, accountCommand.RefundTransfer{AccountID: from, TransferID: transferID, Amount: 300}, refund.TriggeredWithCommand().Payload)
		})
	})
}
