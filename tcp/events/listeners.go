package events

import (
	"fmt"

	"github.com/grafana/sobek"
)

// EventListeners keeps track of the EventListeners for each event type
type EventListeners struct {
	open  *EventListener
	data  *EventListener
	error *EventListener
	close *EventListener
}

// NewEventListeners returns an initial EventListeners struct you need to start with.
func NewEventListeners() *EventListeners {
	return &EventListeners{
		open:  newListener(OPEN),
		data:  newListener(DATA),
		error: newListener(ERROR),
		close: newListener(CLOSE),
	}
}

// EventListener represents a tuple of listeners of a certain type
// property on represents the EventListener that serves for the on* properties, like onopen, onmessage, etc.
// property list keeps any other listeners that were added with addEventListener
type EventListener struct {
	eventType string

	// this return sobek.value *and* error in order to return error on exception instead of panic
	// https://pkg.go.dev/github.com/dop251/goja#hdr-Functions
	on   func(sobek.Value) (sobek.Value, error)
	list []func(sobek.Value) (sobek.Value, error)
}

// newListener creates a new listener of a certain type
func newListener(eventType string) *EventListener {
	return &EventListener{
		eventType: eventType,
	}
}

// Add func adds a listener to the listener list
func (l *EventListener) Add(fn func(sobek.Value) (sobek.Value, error)) {
	l.list = append(l.list, fn)
}

// SetOn func sets a listener for the on* properties, like onopen, onmessage, etc.
func (l *EventListener) SetOn(fn func(sobek.Value) (sobek.Value, error)) {
	l.on = fn
}

// GetOn func returns the on* property for a certain event type
func (l *EventListener) GetOn() func(sobek.Value) (sobek.Value, error) {
	return l.on
}

// All func Return all possible listeners for a certain event type
func (l *EventListener) All() []func(sobek.Value) (sobek.Value, error) {
	if l.on == nil {
		return l.list
	}

	return append([]func(sobek.Value) (sobek.Value, error){l.on}, l.list...)
}

// GetType func return event listener of a certain type
func (l *EventListeners) GetType(t string) *EventListener {
	switch t {
	case OPEN:
		return l.open
	case DATA:
		return l.data
	case ERROR:
		return l.error
	case CLOSE:
		return l.close
	default:
		return nil
	}
}

// Add func adds a listener to the listeners
func (l *EventListeners) Add(t string, f func(sobek.Value) (sobek.Value, error)) error {
	list := l.GetType(t)

	if list == nil {
		return fmt.Errorf("unknown event type: %s", t)
	}

	list.Add(f)

	return nil
}

// All func returns all possible listeners for a certain event type or an empty array
func (l *EventListeners) All(t string) []func(sobek.Value) (sobek.Value, error) {
	list := l.GetType(t)

	if list == nil {
		return []func(sobek.Value) (sobek.Value, error){}
	}

	return list.All()
}
