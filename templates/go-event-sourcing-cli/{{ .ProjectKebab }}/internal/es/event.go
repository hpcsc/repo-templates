package es

import "time"

type Event interface {
	Kind() string
}

type Record struct {
	StreamID string
	Seq      int
	At       time.Time
	Event    Event
}
