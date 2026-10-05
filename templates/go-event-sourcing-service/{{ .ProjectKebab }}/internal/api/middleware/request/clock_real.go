package request

import "time"

func NewRealClock() Clock {
	return &realClock{}
}

type realClock struct{}

func (c realClock) Now() time.Time {
	return time.Now()
}

func (c realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}
