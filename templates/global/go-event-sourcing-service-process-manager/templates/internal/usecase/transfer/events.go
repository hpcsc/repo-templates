package transfer

import (
	"github.com/hpcsc/go-event-sourcing-service-project/internal/domain/event/registry"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/usecase/transfer/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.TransferStarted]()
	reg.Register[event.TransferDebitRequested]()
	reg.Register[event.TransferCreditRequested]()
	reg.Register[event.TransferRefundRequested]()
	reg.Register[event.TransferCompleted]()
	reg.Register[event.TransferFailed]()
}
