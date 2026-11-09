package natspush

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamName = "PUSH"
	Subject    = "push.events"
)

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func EnsureStream(js nats.JetStreamContext) error {
	if _, err := js.StreamInfo(StreamName); err == nil {
		return nil
	} else if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}

	_, err := js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{Subject},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    time.Hour,
		Discard:   nats.DiscardOld,
	})
	if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return nil
	}
	return err
}
