package background

import (
	"context"
	"github.com/rs/zerolog"
	"sync"
)

func NewRunner(logger zerolog.Logger) *Runner {
	return &Runner{
		wg:     &sync.WaitGroup{},
		logger: logger,
	}
}

type Runner struct {
	wg     *sync.WaitGroup
	logger zerolog.Logger
}

func (r *Runner) Run(ctx context.Context, job Job) {
	r.wg.Add(1)

	l := r.logger.With().Str("job", job.Name()).Logger()

	go func() {
		l.Info().Msg("job started")
		defer r.wg.Done()

		job.Run(ctx, l)

		l.Info().Msg("job stopped")
	}()
}

func (r *Runner) Wait() {
	r.wg.Wait()
}
