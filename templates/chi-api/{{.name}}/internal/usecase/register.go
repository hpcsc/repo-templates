package usecase

import (
	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{.name}}/internal/usecase/root"
	"github.com/hpcsc/{{.name}}/internal/usecase/user"
)

func Register(router chi.Router) {
	root.Register(router)
	user.Register(router)
}
