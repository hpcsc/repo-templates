package use_cases

import (
{{- if or .Scaffold.PushSubscriber .Scaffold.PushNATS }}
	"context"
{{- end }}
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/middleware"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
	apiAccount "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/account"
{{- if .Scaffold.ProcessManager }}
	apiPayment "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/payment"
{{- end }}
{{- if or .Scaffold.PushSubscriber .Scaffold.PushNATS }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/websocket"
{{- end }}
{{- if .Scaffold.PushNATS }}
	"github.com/nats-io/nats.go"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/root"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/webhook"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
{{- if and .Scaffold.PushSubscriber (not .Scaffold.PushNATS) }}
	checkpointStore "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	coreUseCases "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/balance"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/summary"
{{- end }}
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(router chi.Router, cfg *api.Config, pool *pgxpool.Pool, logger *slog.Logger) (func(), error) {
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.TokenPath)
	if err != nil {
		return nil, err
	}

{{- if .Scaffold.PushNATS }}
	wsHub := websocket.NewHub(logger)
	natsConn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	bridgeCtx, stopBridge := context.WithCancel(context.Background())
	go func() {
		if err := websocket.BridgeNATS(bridgeCtx, natsConn, wsHub, logger); err != nil {
			logger.Error("nats push bridge failed", "error", err)
		}
	}()
{{- else if .Scaffold.PushSubscriber }}
	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	wsHub := websocket.NewHub(logger)
	checkpoints := checkpointStore.NewPostgres(pool)
	go startPushSubscriber(context.Background(), pool, reg, wsHub, checkpoints, logger)
{{- end }}

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range allRoutes(pool{{- if or .Scaffold.PushSubscriber .Scaffold.PushNATS }}, wsHub{{- end }}) {
		if r.IsPublic() {
			publicRoutes = append(publicRoutes, r)
		} else if r.IsProtected() {
			protectedRoutes = append(protectedRoutes, r)
		}
	}

	router.Route("/api", func(apiRouter chi.Router) {
		for _, r := range publicRoutes {
			apiRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
			logger.Info(fmt.Sprintf("registered public route %s %s", r.Method, r.Pattern))
		}

		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(authMiddleware)

			for _, r := range protectedRoutes {
				protectedRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
				logger.Info(fmt.Sprintf("registered protected route %s %s", r.Method, r.Pattern))
			}
		})
	})

	return {{- if .Scaffold.PushNATS }} func() {
		stopBridge()
		wsHub.Shutdown(context.Background())
		_ = natsConn.Drain()
		natsConn.Close()
	} {{- else if .Scaffold.PushSubscriber }} func() { wsHub.Shutdown(context.Background()) } {{- else }} nil {{- end }}, nil
}

func allRoutes(pool *pgxpool.Pool{{- if or .Scaffold.PushSubscriber .Scaffold.PushNATS }}, wsHub *websocket.Hub{{- end }}) []*route.Route {
	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus(command.WithHistory(eventStream))
	coreUseCases.RegisterHandlers(bus, eventStream)

	routables := []route.Routable{
		root.NewHandler(),
		apiAccount.NewHandler(bus, balance.NewQuery(pool)),
{{- if .Scaffold.ProcessManager }}
		apiPayment.NewHandler(bus, summary.NewQuery(pool)),
		webhook.NewHandler(bus),
{{- end }}
{{- if or .Scaffold.PushSubscriber .Scaffold.PushNATS }}
		websocket.NewHandler(wsHub, slog.Default()),
{{- end }}
	}

	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}