package process

import (
	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/event"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/event"
)

type State string

const (
	Debiting  State = "debiting"
	Crediting State = "crediting"
	Refunding State = "refunding"
	Completed State = "completed"
	Failed    State = "failed"
)

type Process struct {
	transferID    uuid.UUID
	from          uuid.UUID
	to            uuid.UUID
	amount        int64
	state         State
	reason        string
	streamVersion uint64
}

func Start(started *transferEvent.TransferStarted, streamVersion uint64) (*Process, []domain.Event) {
	p := &Process{
		transferID:    started.TransferID,
		from:          started.From,
		to:            started.To,
		amount:        started.Amount,
		state:         Debiting,
		streamVersion: streamVersion,
	}

	return p, []domain.Event{domain.NewEvent(&transferEvent.TransferDebitRequested{
		TransferID: p.transferID,
		AccountID:  p.from,
		Amount:     p.amount,
	})}
}

func (p *Process) State() State {
	return p.state
}

func (p *Process) Reason() string {
	return p.reason
}

func (p *Process) Advance(evt domain.Event) []domain.Event {
	switch e := evt.Payload.(type) {
	case *accountEvent.TransferDebited:
		if p.state != Debiting {
			return nil
		}
		p.state = Crediting
		return []domain.Event{domain.NewEvent(&transferEvent.TransferCreditRequested{
			TransferID: p.transferID,
			AccountID:  p.to,
			Amount:     p.amount,
		})}
	case *accountEvent.TransferDebitRejected:
		if p.state != Debiting {
			return nil
		}
		return p.fail(e.Reason)
	case *accountEvent.TransferCredited:
		if p.state != Crediting {
			return nil
		}
		p.state = Completed
		return []domain.Event{domain.NewEvent(&transferEvent.TransferCompleted{
			TransferID: p.transferID,
		})}
	case *accountEvent.TransferCreditRejected:
		if p.state != Crediting {
			return nil
		}
		p.state = Refunding
		p.reason = e.Reason
		return []domain.Event{domain.NewEvent(&transferEvent.TransferRefundRequested{
			TransferID: p.transferID,
			AccountID:  p.from,
			Amount:     p.amount,
		})}
	case *accountEvent.TransferRefunded:
		if p.state != Refunding {
			return nil
		}
		return p.fail(p.reason)
	}
	return nil
}

func (p *Process) fail(reason string) []domain.Event {
	p.state = Failed
	p.reason = reason
	return []domain.Event{domain.NewEvent(&transferEvent.TransferFailed{
		TransferID: p.transferID,
		Reason:     reason,
	})}
}
