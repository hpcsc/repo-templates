package job

import (
	"context"
	"time"

	"github.com/hpcsc/{{.name}}/internal/background"
	"github.com/rs/zerolog"
)

var _ background.Job = (*mainJob)(nil)

func NewMainJob() background.Job {
	return &mainJob{}
}

type mainJob struct{}

func (j *mainJob) Name() string {
	return "main"
}

func (j *mainJob) Run(ctx context.Context, logger zerolog.Logger) {
	ticker := time.NewTicker(2 * time.Second)

	for ; true; <-ticker.C {
		select {
		case <-ctx.Done():
			logger.Info().Msg("stopping ticker")
			ticker.Stop()
			logger.Info().Msg("ticker stopped")

			return
		default:
			logger.Info().Msg("processing")
		}
	}
}
