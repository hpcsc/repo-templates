package balance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/event"
	"github.com/jackc/pgx/v5"
)

var _ projection.Interface = (*Projection)(nil)

func NewProjection() *Projection {
	return &Projection{}
}

type Projection struct{}

func (p *Projection) Name() string {
	return "account-balances"
}

func (p *Projection) EventTypes() []string {
	return []string{
		event.AccountOpened{}.EventType(),
		event.MoneyDeposited{}.EventType(),
		event.MoneyWithdrawn{}.EventType(),
		event.WelcomeBonusGranted{}.EventType(),
		event.AccountClosed{}.EventType(),
		event.HoldCaptured{}.EventType(),
{{- if .Scaffold.ProcessManager }}
		event.TransferDebited{}.EventType(),
		event.TransferCredited{}.EventType(),
		event.TransferRefunded{}.EventType(),
{{- end }}
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.AccountOpened:
		_, err := tx.Exec(ctx,
			`INSERT INTO account_balances (id, owner, balance, version, updated_at) VALUES ($1, $2, 0, $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			payload.AccountID, payload.Owner, evt.Version, evt.RecordedAt,
		)
		return err
	case *event.MoneyDeposited:
		return p.change(ctx, tx, payload.AccountID, payload.Amount, evt)
	case *event.MoneyWithdrawn:
		return p.change(ctx, tx, payload.AccountID, -payload.Amount, evt)
	case *event.WelcomeBonusGranted:
		return p.change(ctx, tx, payload.AccountID, payload.Amount, evt)
	case *event.AccountClosed:
		_, err := tx.Exec(ctx,
			`UPDATE account_balances SET status = 'closed', version = $2, updated_at = $3
			WHERE id = $1 AND version < $2`,
			payload.AccountID, evt.Version, evt.RecordedAt,
		)
		return err
	case *event.HoldCaptured:
		return p.change(ctx, tx, payload.AccountID, -payload.Amount, evt)
{{- if .Scaffold.ProcessManager }}
	case *event.TransferDebited:
		return p.change(ctx, tx, payload.AccountID, -payload.Amount, evt)
	case *event.TransferCredited:
		return p.change(ctx, tx, payload.AccountID, payload.Amount, evt)
	case *event.TransferRefunded:
		return p.change(ctx, tx, payload.AccountID, payload.Amount, evt)
{{- end }}
	default:
		return fmt.Errorf("received unexpected payload type: %T", evt.Payload)
	}
}

func (p *Projection) change(ctx context.Context, tx pgx.Tx, accountID uuid.UUID, amount int64, evt *domain.Event) error {
	_, err := tx.Exec(ctx,
		`UPDATE account_balances SET balance = balance + $2, version = $3, updated_at = $4
		WHERE id = $1 AND version < $3`,
		accountID, amount, evt.Version, evt.RecordedAt,
	)
	return err
}
