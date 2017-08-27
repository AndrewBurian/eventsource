package eventsource

import (
	"bytes"
	"io"
	"io/ioutil"
	"strconv"
)

// Holds the data for an event
type Event struct {
	id     []byte
	data   [][]byte
	event  []byte
	retry  uint64
	buf    bytes.Buffer
	bufSet bool
}

// DataEvent creates a new Event with the data field set
func DataEvent(data string) *Event {
	e := &Event{}
	io.WriteString(e, data)
	return e
}

// ID sets the event ID
func (e *Event) ID(id string) {
	e.id = []byte(id)
}

// Read the event in wire format
func (e *Event) Read(p []byte) (int, error) {
	if e.bufSet {
		return e.buf.Read(p)
	}

	// Wipe out any existing data
	e.buf.Reset()

	// event:
	if len(e.event) > 0 {
		e.buf.WriteString("event: ")
		e.buf.Write(e.event)
		e.buf.WriteByte('\n')
	}

	// id:
	if len(e.id) > 0 {
		e.buf.WriteString("id: ")
		e.buf.Write(e.id)
		e.buf.WriteByte('\n')
	}

	// data:
	if len(e.data) > 0 {
		for _, entry := range e.data {
			e.buf.WriteString("data: ")
			e.buf.Write(entry)
			e.buf.WriteByte('\n')
		}
	}

	// retry:
	if e.retry > 0 {
		e.buf.WriteString("retry: ")
		e.buf.WriteString(strconv.FormatUint(e.retry, 10))
		e.buf.WriteByte('\n')
	}

	e.buf.WriteByte('\n')
	e.bufSet = true

	return e.buf.Read(p)
}

// Write to the event. Buffer will be converted to one or more
// `data` sections in wire format
//
// Successive calls to write will each create data entry lines
//
// Newlines will be split into multiple data entry lines, successive
// newlines are discarded
func (e *Event) Write(p []byte) (int, error) {
	// split event on newlines
	split := bytes.Split(p, []byte{'\n'})
	for _, entry := range split {
		// don't write empty entries
		if len(entry) == 0 {
			continue
		}
		e.data = append(e.data, entry)
	}
	e.bufSet = false
	return len(p), nil
}

// WriteRaw sets an event directly in wire format
//
// This does no validation to ensure it is in a correct format
// and should mostly be used to deep copy another event
func (e *Event) WriteRaw(p []byte) (int, error) {
	return e.buf.Write(p)
}

// String returns the Event in wire format as a string
func (e *Event) String() string {
	fullEvent, _ := ioutil.ReadAll(e)
	return string(fullEvent)
}
