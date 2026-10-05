package fake

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
)

var _ command.Handler[Payload] = (*Handler[Payload])(nil)

func NewHandler[P any]() *Handler[P] {
	return &Handler[P]{}
}

type Handler[P any] struct {
	commands []domain.Command[P]
	errs     []error
}

func (h *Handler[P]) WithError(err error) *Handler[P] {
	h.errs = []error{err}
	return h
}

func (h *Handler[P]) WithErrors(errs ...error) *Handler[P] {
	h.errs = errs
	return h
}

func (h *Handler[P]) Handle(_ context.Context, cmd domain.Command[P]) error {
	h.commands = append(h.commands, cmd)
	if len(h.errs) == 0 {
		return nil
	}

	err := h.errs[0]
	if len(h.errs) > 1 {
		h.errs = h.errs[1:]
	}
	return err
}

func (h *Handler[P]) TriggeredWithCommand() *domain.Command[P] {
	if len(h.commands) == 0 {
		return nil
	}
	return &h.commands[len(h.commands)-1]
}

func (h *Handler[P]) Calls() int {
	return len(h.commands)
}
