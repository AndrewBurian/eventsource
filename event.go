package eventsource

import (
	"bytes"
	"strconv"
	"strings"
)

// Event holds the structured data for an event.
// The event object contains working memory as well as a bytes
// buffer. The buffer is filled from the working area at the first
// call to either Read or String. Mutating the event resets the buffer
// but sequential calls to Read do not.
type Event struct {
	id     string
	data   []string
	event  string
	retry  uint64
	buf    bytes.Buffer
	bufSet bool
}

// ID sets the event ID
func (e *Event) ID(id string) *Event {
	e.id = id
	e.bufSet = false
	return e
}

// Type sets the event's event: field
func (e *Event) Type(t string) *Event {
	e.event = t
	e.bufSet = false
	return e
}

// Retry sets the event's retry: field
func (e *Event) Retry(t uint64) *Event {
	e.retry = t
	e.bufSet = false
	return e
}

// Data replaces the data with the given string
func (e *Event) Data(dat string) *Event {
	// truncate
	e.data = e.data[:0]
	e.WriteString(dat)
	return e
}

// AppendData adds data to the event without overwriting
func (e *Event) AppendData(dat string) *Event {
	e.WriteString(dat)
	return e
}

// Read the event in wire format
func (e *Event) Read(p []byte) (int, error) {
	if e.bufSet {
		return e.buf.Read(p)
	}

	e.prepare()

	return e.buf.Read(p)
}

// Prepares the data buf for reading
func (e *Event) prepare() {

	// Wipe out any existing data
	e.buf.Reset()

	// event:
	if len(e.event) > 0 {
		e.buf.WriteString("event: ")
		e.buf.WriteString(e.event)
		e.buf.WriteByte('\n')
	}

	// id:
	if len(e.id) > 0 {
		e.buf.WriteString("id: ")
		e.buf.WriteString(e.id)
		e.buf.WriteByte('\n')
	}

	// data:
	if len(e.data) > 0 {
		for _, entry := range e.data {
			e.buf.WriteString("data: ")
			e.buf.WriteString(entry)
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
}

// Write to the event. Buffer will be converted to one or more
// `data` sections in wire format
//
// Successive calls to write will each create data entry lines
//
// Newlines will be split into multiple data entry lines, successive
// newlines are discarded
func (e *Event) Write(p []byte) (int, error) {
	e.WriteString(string(p))
	return len(p), nil
}

// WriteString adds string data to the event.
// Equivalent to calling Write([]byte(string))
func (e *Event) WriteString(p string) {
	// split event on newlines
	split := strings.Split(p, "\n")
	for _, entry := range split {
		// don't write empty entries
		if len(entry) == 0 {
			continue
		}
		e.data = append(e.data, entry)
	}
	e.bufSet = false
}

// WriteRaw sets an event directly in wire format
//
// This does no validation to ensure it is in a correct format
// and should mostly be used to deep copy another event
func (e *Event) WriteRaw(p []byte) (int, error) {
	e.bufSet = true
	return e.buf.Write(p)
}

// String returns the Event in wire format as a string
func (e *Event) String() string {
	e.prepare()
	return string(e.buf.Bytes())
}

// Clone returns a deep copy of the event
func (e *Event) Clone() *Event {
	clone := &Event{
		id:    e.id,
		event: e.event,
		retry: e.retry,
	}

	clone.data = append(clone.data, e.data...)
	return clone
}
