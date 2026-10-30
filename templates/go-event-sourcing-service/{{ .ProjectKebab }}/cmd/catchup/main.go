package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/config"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/lock"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/scheduler"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/balance"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/welcomebonus"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/checks"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/deadlines"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/funding"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/holds"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/outside"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/status"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/submission"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/summary"
{{- end }}
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
{{- if .Scaffold.ProcessManager }}
	DB      config.DB
	Outside outside.Config
{{- else }}
	DB config.DB
{{- end }}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load[Config]()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DB.ConnectionString())
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctx := runShutdownSignalHandler(context.Background(), logger)
	failed := run(ctx, workers(pool, cfg, logger), lock.NewPostgres(pool), logger)

	pool.Close()
	if failed {
		os.Exit(1)
	}
}

// worker is a consumer or a scheduler. catchup runs each one under the lock of its name.
type worker interface {
	Name() string
	Start(ctx context.Context) error
}

func workers(pool *pgxpool.Pool, cfg *Config, logger *slog.Logger) []worker {
	reg := registry.New()
	use_cases.RegisterEvents(reg)

	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus(command.WithHistory(eventStream))
	use_cases.RegisterHandlers(bus, eventStream)

	checkpointStore := store.NewPostgres(pool)
	eventSubscription := subscription.NewPostgres(pool, reg, logger)

	return []worker{
		projection.NewProjector(balance.NewProjection(), pool, checkpointStore, eventSubscription, logger),
		reaction.NewReactor(welcomebonus.NewReaction(bus), checkpointStore, eventSubscription, logger),
{{- if .Scaffold.ProcessManager }}
		reaction.NewReactor(funding.NewReaction(bus), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
		reaction.NewReactor(holds.NewReaction(bus), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
		reaction.NewReactor(checks.NewGateway(payment.NewRepository(eventStream), outside.NewClient(), cfg.Outside, logger), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
		reaction.NewReactor(submission.NewGateway(bus, payment.NewRepository(eventStream), outside.NewClient(), cfg.Outside, logger), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
		reaction.NewReactor(status.NewGateway(bus, payment.NewRepository(eventStream), outside.NewClient(), cfg.Outside, logger), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
		projection.NewProjector(summary.NewProjection(), pool, checkpointStore, eventSubscription, logger),
		projection.NewProjector(deadlines.NewProjection(), pool, checkpointStore, eventSubscription, logger),
		scheduler.New(deadlines.NewReporter(pool, bus), logger),
{{- end }}
	}
}

func run(parentCtx context.Context, workers []worker, locks *lock.Postgres, logger *slog.Logger) bool {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var failed atomic.Bool
	var wg sync.WaitGroup
	for _, w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					logger.Error(fmt.Sprintf("%s received unhandled error: %v", w.Name(), err))
					failed.Store(true)
					cancel()
				}
			}()

			logger.Info(fmt.Sprintf("%s takes its lock", w.Name()))
			if err := locks.Hold(ctx, w.Name(), w.Start); err != nil {
				logger.Error(fmt.Sprintf("%s failed: %v", w.Name(), err))
				failed.Store(true)
				cancel()
				return
			}
			logger.Info(fmt.Sprintf("%s stopped", w.Name()))
		}()
	}

	wg.Wait()
	return failed.Load()
}

func runShutdownSignalHandler(parentCtx context.Context, logger *slog.Logger) context.Context {
	ctx, cancel := context.WithCancel(parentCtx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))
		cancel()
	}()

	return ctx
}
