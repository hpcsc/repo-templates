//go:build unit || integration

package projection_test

import "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain"

func closedChannel(events ...domain.Event) <-chan domain.Event {
	ch := make(chan domain.Event, len(events))
	for _, evt := range events {
		ch <- evt
	}
	close(ch)
	return ch
}
