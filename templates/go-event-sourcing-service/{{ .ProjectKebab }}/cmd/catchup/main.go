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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/consumer"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/subscription"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/projection"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/reaction"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/balance"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/welcomebonus"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
{{- end }}
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DB config.DB
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
	failed := run(ctx, consumers(pool, logger), logger)

	pool.Close()
	if failed {
		os.Exit(1)
	}
}

func consumers(pool *pgxpool.Pool, logger *slog.Logger) []*consumer.Consumer {
	reg := registry.New()
	usecase.RegisterEvents(reg)

	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus()
	usecase.RegisterHandlers(bus, eventStream)

	checkpointStore := store.NewPostgres(pool)
	eventSubscription := subscription.NewPostgres(pool, reg, logger)

	return []*consumer.Consumer{
		projection.NewProjector(balance.NewProjection(), pool, checkpointStore, eventSubscription, logger),
		reaction.NewReactor(welcomebonus.NewReaction(bus), checkpointStore, eventSubscription, logger),
{{- if .Scaffold.ProcessManager }}
		projection.NewProjector(process.NewProjection(eventStream), pool, checkpointStore, eventSubscription, logger),
		reaction.NewReactor(process.NewDispatcher(bus), checkpointStore, eventSubscription, logger, reaction.StartAtBeginning()),
{{- end }}
	}
}

func run(parentCtx context.Context, consumers []*consumer.Consumer, logger *slog.Logger) bool {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var failed atomic.Bool
	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					logger.Error(fmt.Sprintf("consumer %s received unhandled error: %v", c.Name(), err))
					failed.Store(true)
					cancel()
				}
			}()

			if err := c.Start(ctx); err != nil {
				logger.Error(fmt.Sprintf("consumer %s failed: %v", c.Name(), err))
				failed.Store(true)
				cancel()
				return
			}
			logger.Info(fmt.Sprintf("consumer %s stopped", c.Name()))
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
