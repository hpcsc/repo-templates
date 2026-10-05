package command

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

type Handler[P any] interface {
	Handle(ctx context.Context, cmd domain.Command[P]) error
}
