package reaction

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

type Interface interface {
	Handle(ctx context.Context, event *domain.Event) error
	EventTypes() []string
	Name() string
}
