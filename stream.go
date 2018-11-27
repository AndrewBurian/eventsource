/*
Package eventsource is a library for dealing with server sent events in Go.

The library attempts to make as few assumptions as possible about how your apps
will be written, and remains as generic as possible while still being simple and useful.

The three core objects to the library are Clients, Events, and Streams.

Client wraps an HTTP connection, and runs a worker routine to send events to the connected
client in a thread-safe way. It gracefully handles client disconnects.

Event encapsulates all the data that can be sent with SSE, and contains the logic for converting
it to wire format to send. Events are not thread-safe by themselves.

Stream is an abstraction for 0 or more Client connections, and adds some multiplexing and filtering
on top of the Client. It also can act as an http.Handler to automatically register inbound client
connections and disconnections.

A quick example of a simple sever that broadcasts a "tick" event every second

	func main() {
		stream := eventsource.NewStream()
		go func(s *eventsource.Stream) {
			for {
				time.Sleep(time.Second)
				stream.Broadcast(eventsource.DataEvent("tick"))
			}
		}(stream)
		http.ListenAndServe(":8080", stream)
	}

*/
package eventsource

import (
	"net/http"
	"sync"
)

// Stream abstracts several client connections together and allows
// for event multiplexing and topics.
// A stream also implements an http.Handler to easily register incoming
// http requests as new clients.
type Stream struct {
	clients           map[*Client]topicList
	listLock          sync.RWMutex
	shutdownWait      sync.WaitGroup
	clientConnectHook func(*http.Request, *Client)
}

type topicList map[string]bool

// NewStream creates a new stream object
func NewStream() *Stream {
	return &Stream{
		clients: make(map[*Client]topicList),
	}
}

// Register adds a client to the stream to receive all broadcast
// messages. Has no effect if the client is already registered.
func (s *Stream) Register(c *Client) {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	// see if the client has been registered
	if _, found := s.clients[c]; found {
		return
	}

	// append new client
	s.clients[c] = make(topicList)
}

// Remove will remove a client from this stream, but not shut the client down.
func (s *Stream) Remove(c *Client) {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	delete(s.clients, c)
}

// Broadcast sends the event to all clients registered on this stream.
func (s *Stream) Broadcast(e *Event) {
	s.listLock.RLock()
	defer s.listLock.RUnlock()

	for cli := range s.clients {
		cli.Send(e)
	}
}

// Subscribe add the client to the list of clients receiving publications
// to this topic. Subscribe will also Register an unregistered
// client.
func (s *Stream) Subscribe(topic string, c *Client) {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	// see if the client is registered
	topics, found := s.clients[c]

	// register if not
	if !found {
		topics = make(topicList)
		s.clients[c] = topics
	}

	topics[topic] = true
}

// Unsubscribe removes clients from the topic, but not from broadcasts.
func (s *Stream) Unsubscribe(topic string, c *Client) {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	topics, found := s.clients[c]
	if !found {
		return
	}
	topics[topic] = false
}

// Publish sends the event to clients that have subscribed to the given topic.
func (s *Stream) Publish(topic string, e *Event) {
	s.listLock.RLock()
	defer s.listLock.RUnlock()

	for cli, topics := range s.clients {
		if topics[topic] {
			cli.Send(e)
		}
	}
}

// Shutdown terminates all clients connected to the stream and removes them
func (s *Stream) Shutdown() {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	for client := range s.clients {
		client.Shutdown()
		delete(s.clients, client)
	}
}

// CloseTopic removes all client associations with this topic, but does not
// terminate them or remove
func (s *Stream) CloseTopic(topic string) {

}

// ServeHTTP takes a client connection, registers it for broadcasts,
// then blocks so long as the connection is alive.
func (s *Stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// ensure the client accepts an event-stream
	if !checkRequest(r) {
		http.Error(w, "This is an EventStream endpoint", http.StatusNotAcceptable)
		return
	}

	// create the client
	c := NewClient(w, r)
	if c == nil {
		http.Error(w, "EventStream not supported for this connection", http.StatusInternalServerError)
		return
	}

	// wait for the client to exit or be shutdown
	s.Register(c)
	if s.clientConnectHook != nil {
		s.clientConnectHook(r, c)
	}
	c.Wait()
	s.Remove(c)
}

// TopicHandler returns an HTTP handler that will register a client for broadcasts
// and for any topics, and then block so long as they are connected
func (s *Stream) TopicHandler(topics []string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		// ensure the client accepts an event-stream
		if !checkRequest(r) {
			http.Error(w, "This is an EventStream endpoint", http.StatusNotAcceptable)
			return
		}

		// create the client
		c := NewClient(w, r)
		if c == nil {
			http.Error(w, "EventStream not supported for this connection", http.StatusInternalServerError)
			return
		}

		// broadcasts
		s.Register(c)

		// topics
		for _, topic := range topics {
			s.Subscribe(topic, c)
		}

		if s.clientConnectHook != nil {
			s.clientConnectHook(r, c)
		}

		// wait for the client to exit or be shutdown
		c.Wait()
		s.Remove(c)
	}
}

// ClientConnectHook sets a function to be called when a client connects to this stream's
// HTTP handler.
// Only one handler may be registered. Further calls overwrite the previous.
func (s *Stream) ClientConnectHook(fn func(*http.Request, *Client)) {
	s.clientConnectHook = fn
}

// NumClients returns the number of currently connected clients
func (s *Stream) NumClients() int {
	return len(s.clients)
}

// Checks that a client expects an event-stream
func checkRequest(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/event-stream"
}
