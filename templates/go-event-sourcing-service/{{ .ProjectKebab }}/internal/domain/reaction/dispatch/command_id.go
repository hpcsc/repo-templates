package dispatch

import (
	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

func CommandIDFor(reactionName string, event *domain.Event) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(reactionName+"/"+event.ID))
}
