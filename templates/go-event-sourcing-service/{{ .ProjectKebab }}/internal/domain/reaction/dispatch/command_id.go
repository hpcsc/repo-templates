package dispatch

import (
	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"
)

func CommandIDFor(reactionName string, event *domain.Event) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(reactionName+"/"+event.ID))
}

func CommandFor[P any](reactionName string, event *domain.Event, payload P) domain.Command[P] {
	caused := event.CausedMetadata()
	return domain.Command[P]{
		ID:            CommandIDFor(reactionName, event),
		CorrelationID: caused.CorrelationID,
		CausationID:   caused.CausationID,
		Payload:       payload,
	}
}
