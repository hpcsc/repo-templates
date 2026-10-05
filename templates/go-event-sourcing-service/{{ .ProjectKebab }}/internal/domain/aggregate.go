package domain

type Aggregate interface {
	Apply(event Event)
	Recorded() []Event
	Version() uint64
	MarkCommitted(version uint64)
}
