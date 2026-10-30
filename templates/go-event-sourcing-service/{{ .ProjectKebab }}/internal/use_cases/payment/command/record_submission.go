package command

import "github.com/google/uuid"

type RecordSubmission struct {
	PaymentID        uuid.UUID
	NetworkReference string
}
