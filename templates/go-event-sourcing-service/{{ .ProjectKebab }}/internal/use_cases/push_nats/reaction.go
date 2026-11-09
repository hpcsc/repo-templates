package push_nats

import (
	"context"
	"encoding/json"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/natspush"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/nats-io/nats.go"
)

var _ reaction.Interface = (*Reaction)(nil)

type Reaction struct {
	js nats.JetStreamContext
}

func New(conn *nats.Conn) (*Reaction, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}
	if err := natspush.EnsureStream(js); err != nil {
		return nil, err
	}
	return &Reaction{js: js}, nil
}

const Name = "push-nats"

func (r *Reaction) Name() string {
	return Name
}

func (r *Reaction) EventTypes() []string {
	return nil
}

func (r *Reaction) Handle(_ context.Context, evt *domain.Event) error {
	body, err := json.Marshal(natspush.Message{Type: evt.Type(), Data: evt.Payload})
	if err != nil {
		return err
	}

	_, err = r.js.Publish(natspush.Subject, body)
	return err
}
