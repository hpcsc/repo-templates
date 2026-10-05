package domain

type AggregateBase struct {
	version  uint64
	recorded []Event
}

func (b *AggregateBase) Record(event Event) {
	b.recorded = append(b.recorded, event)
}

func (b *AggregateBase) Recorded() []Event {
	return b.recorded
}

func (b *AggregateBase) Version() uint64 {
	return b.version
}

func (b *AggregateBase) MarkCommitted(version uint64) {
	b.version = version
	b.recorded = nil
}
