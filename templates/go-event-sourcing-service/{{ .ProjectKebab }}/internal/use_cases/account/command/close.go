package command

import "github.com/google/uuid"

type Close struct {
	AccountID uuid.UUID
}
