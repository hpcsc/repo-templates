// Package es is the event-sourcing foundation: the command and event
// contracts, the event store, the codec that persists events, and the
// aggregate repository that ties them together.
//
// It knows nothing about any particular domain and depends on nothing but the
// standard library. Everything under internal/ that is not this package is
// example code you are expected to delete.
package es
