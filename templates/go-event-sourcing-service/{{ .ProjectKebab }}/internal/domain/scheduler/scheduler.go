package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const DefaultInterval = time.Second

// Tick sends commands with a fixed ID for each due item, because a tick that fails runs again.
type Interface interface {
	Name() string
	Tick(ctx context.Context, now time.Time) error
}

type Option func(*Scheduler)

func WithInterval(interval time.Duration) Option {
	return func(s *Scheduler) {
		s.interval = interval
	}
}

func New(scheduled Interface, logger *slog.Logger, opts ...Option) *Scheduler {
	s := &Scheduler{
		scheduled: scheduled,
		interval:  DefaultInterval,
		logger:    logger.With("scheduler", scheduled.Name()),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Scheduler struct {
	scheduled Interface
	interval  time.Duration
	logger    *slog.Logger
}

func (s *Scheduler) Name() string {
	return s.scheduled.Name()
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("starting", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if err := s.scheduled.Tick(ctx, now); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				s.logger.Error("failed to handle a tick", "error", err)
				return fmt.Errorf("scheduler %s failed to handle the tick at %s: %w", s.Name(), now.Format(time.RFC3339), err)
			}
		}
	}
}
