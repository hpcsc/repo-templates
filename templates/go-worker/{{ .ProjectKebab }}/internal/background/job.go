package background

import (
	"context"
	"github.com/rs/zerolog"
)

type Job interface {
	Name() string
	Run(ctx context.Context, logger zerolog.Logger)
}
