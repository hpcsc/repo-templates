package usecase

import (
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/config"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/middleware"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/route"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/usecase/root"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/usecase/user"
)

var routables = []route.Routable{
	root.NewHandler(),
	user.NewHandler(),
}

func Register(router chi.Router, cfg *config.Config, logger *slog.Logger) error {
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.TokenPath)
	if err != nil {
		return err
	}

	routes := allRoutes()

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range routes {
		if r.IsPublic() {
			publicRoutes = append(publicRoutes, r)
		} else if r.IsProtected() {
			protectedRoutes = append(protectedRoutes, r)
		}
	}

	for _, r := range publicRoutes {
		router.MethodFunc(r.Method, r.Pattern, r.Handler)
		logger.Info(fmt.Sprintf("registered public route %s %s", r.Method, r.Pattern))
	}

	router.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(authMiddleware)

		for _, r := range protectedRoutes {
			protectedRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
			logger.Info(fmt.Sprintf("registered protected route %s %s", r.Method, r.Pattern))
		}
	})

	return nil
}

func allRoutes() []*route.Route {
	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}
