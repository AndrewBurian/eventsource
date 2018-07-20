package eventsource

import (
	"io"
	"net/http"
	"sync"
)

// Client wraps an http connection and converts it to an
// event stream.
type Client struct {
	flush  http.Flusher
	write  io.Writer
	close  http.CloseNotifier
	events chan *Event
	closed bool
	waiter sync.WaitGroup
}

// NewClient creates a client wrapping a response writer.
// The response writer must support http.Flusher and http.CloseNotifier
// interfaces.
// When writing, the client will automatically send some headers. Passing the
// original http.Request helps determine which headers, but the request it is
// optional.
// Returns nil on error.
func NewClient(w http.ResponseWriter, req *http.Request) *Client {
	c := &Client{
		events: make(chan *Event, 1),
		write:  w,
	}

	// Check to ensure we support flushing
	flush, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	c.flush = flush

	// Check to ensure we support close notifications
	closer, ok := w.(http.CloseNotifier)
	if !ok {
		return nil
	}
	c.close = closer

	// Send the initial headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	if req == nil || req.ProtoMajor < 2 {
		w.Header().Set("Connection", "keep-alive")
	}
	flush.Flush()

	// start the sending thread
	c.waiter.Add(1)
	go c.run()
	return c
}

// Send queues an event to be sent to the client.
// This does not block until the event has been sent.
// Returns an error if the Client has disconnected
func (c *Client) Send(ev *Event) error {
	if c.closed {
		return io.ErrClosedPipe
	}
	c.events <- ev.Clone()
	return nil
}

// Shutdown terminates a client connection
func (c *Client) Shutdown() {
	close(c.events)
	c.waiter.Wait()
}

// Wait blocks and waits for the client to be shutdown.
// Call this in http handler threads to prevent the server from closing
// the client connection.
func (c *Client) Wait() {
	c.waiter.Wait()
}

// Worker thread for the client responsible for writing events
func (c *Client) run() {

	for {
		select {
		case ev, ok := <-c.events:
			// check for shutdown
			if !ok {
				c.closed = true
				c.waiter.Done()
				return
			}

			// send the event
			io.Copy(c.write, ev)
			c.flush.Flush()

		case _ = <-c.close.CloseNotify():
			c.closed = true
			c.waiter.Done()
			return
		}

	}
}
