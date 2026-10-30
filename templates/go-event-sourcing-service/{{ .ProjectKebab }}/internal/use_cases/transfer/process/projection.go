package process

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	accountEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer"
	transferEvent "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/transfer/event"
	"github.com/jackc/pgx/v5"
)

var _ projection.Interface = (*Projection)(nil)

func NewProjection(stream event.Stream) *Projection {
	return &Projection{
		stream: stream,
	}
}

type Projection struct {
	stream event.Stream
}

func (p *Projection) Name() string {
	return "transfer-process"
}

func (p *Projection) EventTypes() []string {
	return []string{
		transferEvent.TransferStarted{}.EventType(),
		accountEvent.TransferDebited{}.EventType(),
		accountEvent.TransferDebitRejected{}.EventType(),
		accountEvent.TransferCredited{}.EventType(),
		accountEvent.TransferCreditRejected{}.EventType(),
		accountEvent.TransferRefunded{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	if started, ok := evt.Payload.(*transferEvent.TransferStarted); ok {
		return p.start(ctx, tx, evt, started)
	}

	transferID, err := transferIDOf(evt)
	if err != nil {
		return err
	}

	process, err := p.load(ctx, tx, transferID)
	if err != nil {
		return err
	}
	if process == nil {
		return fmt.Errorf("no process for transfer %s", transferID)
	}

	return p.save(ctx, tx, evt, process, process.Advance(*evt))
}

func (p *Projection) start(ctx context.Context, tx pgx.Tx, evt *domain.Event, started *transferEvent.TransferStarted) error {
	existing, err := p.load(ctx, tx, started.TransferID)
	if err != nil || existing != nil {
		return err
	}

	process, requests := Start(started, evt.Version)
	return p.save(ctx, tx, evt, process, requests)
}

func (p *Projection) save(ctx context.Context, tx pgx.Tx, evt *domain.Event, process *Process, requests []domain.Event) error {
	caused := evt.CausedMetadata()
	for i := range requests {
		requests[i].Metadata = &caused
	}

	if err := p.stream.SaveInTx(ctx, tx, transfer.StreamID(process.transferID), requests, process.streamVersion); err != nil {
		return err
	}
	process.streamVersion += uint64(len(requests))

	_, err := tx.Exec(ctx,
		`INSERT INTO transfer_processes (id, from_account, to_account, amount, state, reason, stream_version, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET state = $5, reason = $6, stream_version = $7, updated_at = $8`,
		process.transferID, process.from, process.to, process.amount, process.state, process.reason, process.streamVersion, evt.RecordedAt,
	)
	return err
}

func (p *Projection) load(ctx context.Context, tx pgx.Tx, transferID uuid.UUID) (*Process, error) {
	process := Process{transferID: transferID}
	err := tx.QueryRow(ctx,
		"SELECT from_account, to_account, amount, state, reason, stream_version FROM transfer_processes WHERE id = $1",
		transferID,
	).Scan(&process.from, &process.to, &process.amount, &process.state, &process.reason, &process.streamVersion)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load the process of transfer %s: %w", transferID, err)
	}
	return &process, nil
}

func transferIDOf(evt *domain.Event) (uuid.UUID, error) {
	switch e := evt.Payload.(type) {
	case *accountEvent.TransferDebited:
		return e.TransferID, nil
	case *accountEvent.TransferDebitRejected:
		return e.TransferID, nil
	case *accountEvent.TransferCredited:
		return e.TransferID, nil
	case *accountEvent.TransferCreditRejected:
		return e.TransferID, nil
	case *accountEvent.TransferRefunded:
		return e.TransferID, nil
	default:
		return uuid.Nil, fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}
