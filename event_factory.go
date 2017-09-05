package eventsource

import (
	"io"
	"strconv"
)

// EventFactory is a type of object that can create new events
type EventFactory interface {
	New() *Event
}

// EventIDFactory is an event factory that creates events with
// sequential ID fields.
// If NewFunc is set, the factory uses it to create events before setting
// their IDs
// If NewFunc is not set, NewFact will be used. If neither is set, a new
// event is created from scratch
type EventIDFactory struct {
	NewFact EventFactory
	NewFunc func() *Event
	Next    uint64
}

// New creates an event with the Next id in the sequence
func (f *EventIDFactory) New() *Event {
	var e *Event
	if f.NewFunc != nil {
		e = f.NewFunc()
	} else if f.NewFact != nil {
		e = f.NewFact.New()
	} else {
		e = &Event{}
	}

	e.id = strconv.FormatUint(f.Next, 10)
	f.Next++
	return e
}

// EventTypeFactory creates events of a specific type
type EventTypeFactory struct {
	NewFact EventFactory
	NewFunc func() *Event
	Type    string
}

// New creates an event with the event type set
// If NewFunc is set, the factory uses it to create events before setting
// their event types
// If NewFunc is not set, NewFact will be used. If neither is set, a new
// event is created from scratch
func (f *EventTypeFactory) New() *Event {
	var e *Event
	if f.NewFunc != nil {
		e = f.NewFunc()
	} else if f.NewFact != nil {
		e = f.NewFact.New()
	} else {
		e = &Event{}
	}

	e.event = f.Type
	return e
}

// DataEvent creates a new Event with the data field set
func DataEvent(data string) *Event {
	e := &Event{}
	io.WriteString(e, data)
	return e
}

// TypeEvent creates a new Event with the event field set
func TypeEvent(t string) *Event {
	return &Event{
		event: t,
	}
}
