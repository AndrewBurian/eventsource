package eventsource

import (
	"container/list"
	"net/http"
	"sync"
)

type Stream struct {
	clients           list.List
	broadcast         chan *Event
	shutdownWait      sync.WaitGroup
	clientConnectHook func(*http.Request, *Client)
}

func New() *Stream {
	s := &Stream{}
	go s.run()
	return s
}

func (s *Stream) Register(c *Client) {

}

func (s *Stream) Broadcast(e *Event) {

}

func (s *Stream) Subscribe(topic string, c *Client) {

}

func (s *Stream) Publish(topic string, e *Event) {

}

func (s *Stream) Shutdown() {

}

func (s *Stream) CloseTopic(topic string) {

}

func (s *Stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

func (s *Stream) TopicHandler(topic string) http.HandlerFunc {

}

// Register a function to be called when a client connects to this stream's
// HTTP handler
func (s *Stream) ClientConnectHook(fn func(*http.Request, *Client)) {
	s.clientConnectHook = fn
}

func (s *Stream) run() {

	for {
		select {

		case ev, ok := <-s.broadcast:

			// end of the broadcast channel indicates a shutdown
			if !ok {
				//s.closeAll()
				s.shutdownWait.Done()
				return
			}

			// otherwise normal message
			s.sendAll(ev)

		}
	}
}

func (s *Stream) sendAll(ev *Event) {

}
