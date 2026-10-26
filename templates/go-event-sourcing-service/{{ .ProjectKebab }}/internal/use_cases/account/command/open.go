package command

import "github.com/google/uuid"

type Open struct {
	AccountID uuid.UUID
	Owner     string
}
