package stream

import (
	"context"
	"errors"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/es"
)

var ErrVersionConflict = errors.New("the stream changed after it was loaded")

type Store interface {
	Load(ctx context.Context, streamID string) (records []es.Record, seq int, err error)
	Append(ctx context.Context, streamID string, expectedSeq int, events ...es.Event) error
}

type Projection interface {
	Apply(record es.Record)
}

func Fold(records []es.Record, into ...Projection) {
	for _, record := range records {
		for _, projection := range into {
			projection.Apply(record)
		}
	}
}
