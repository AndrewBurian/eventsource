package eventsource

import (
	"container/list"
	"net/http"
	"sync"
)

type Stream struct {
	clients           list.List
	listLock          sync.RWMutex
	shutdownWait      sync.WaitGroup
	clientConnectHook func(*http.Request, *Client)
}

type registeredClient struct {
	c      *Client
	topics map[string]bool
}

// Register adds a client to the stream to receive all broadcast
// messages. Has no effect if the client is already registered.
func (s *Stream) Register(c *Client) {

	// see if the client has been registered
	if cli := s.getClient(c); cli != nil {
		return
	}

	// append new client
	s.addClient(c)
}

// Remove will remove a client from this stream, but not shut the client down.
func (s *Stream) Remove(c *Client) {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	for element := s.clients.Front(); element != nil; element = element.Next() {
		if regCli := element.Value.(*registeredClient); regCli.c == c {
			// client found
			s.clients.Remove(element)
			return
		}
	}
}

// Broadcast sends the event to all clients registered on this stream.
func (s *Stream) Broadcast(e *Event) {
	s.listLock.RLock()
	defer s.listLock.RUnlock()

	for element := s.clients.Front(); element != nil; element = element.Next() {
		cli := element.Value.(*registeredClient)
		cli.c.Send(e)
	}
}

// Subscribe add the client to the list of clients receiving publications
// to this topic. Subscribe will also Register an unregistered
// client.
func (s *Stream) Subscribe(topic string, c *Client) {
	// see if the client is registered
	cli := s.getClient(c)

	// register if not
	if cli == nil {
		cli = s.addClient(c)
	}

	cli.topics[topic] = true
}

// Unsubscribe removes clients from the topic, but not from broadcasts.
func (s *Stream) Unsubscribe(topic string, c *Client) {

	cli := s.getClient(c)
	if cli == nil {
		return
	}
	cli.topics[topic] = false
}

// Publish sends the event to clients that have subscribed to the given topic.
func (s *Stream) Publish(topic string, e *Event) {
	s.listLock.RLock()
	defer s.listLock.Unlock()

	for element := s.clients.Front(); element != nil; element = element.Next() {
		cli := element.Value.(*registeredClient)
		if cli.topics[topic] {
			cli.c.Send(e)
		}
	}
}

// Shutdown terminates all clients connected to the stream and removes them
func (s *Stream) Shutdown() {
	s.listLock.Lock()
	defer s.listLock.Unlock()

	for element := s.clients.Front(); element != nil; element = element.Next() {
		cli := element.Value.(*registeredClient)
		cli.c.Shutdown()
		s.clients.Remove(element)
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
	c := NewClient(w)
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
		c := NewClient(w)
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

// Register a function to be called when a client connects to this stream's
// HTTP handler
func (s *Stream) ClientConnectHook(fn func(*http.Request, *Client)) {
	s.clientConnectHook = fn
}

// Checks that a client expects an event-stream
func checkRequest(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/event-stream"
}

func (s *Stream) getClient(c *Client) *registeredClient {
	s.listLock.RLock()
	defer s.listLock.RUnlock()

	for element := s.clients.Front(); element != nil; element = element.Next() {
		if regCli := element.Value.(*registeredClient); regCli.c == c {
			// client found
			return regCli
		}
	}

	// not found
	return nil
}

func (s *Stream) addClient(c *Client) *registeredClient {

	cli := &registeredClient{
		c:      c,
		topics: make(map[string]bool),
	}

	s.listLock.Lock()
	s.clients.PushBack(cli)
	s.listLock.Unlock()

	return cli
}
