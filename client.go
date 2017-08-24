package eventsource

import (
	"io"
	"net/http"
)

type Client struct {
	flush http.Flusher
	write io.Writer
}

func NewClient(w http.ResponseWriter) *Client {
	c := &Client{
		write: w,
	}

	// Check to ensure we support flushing
	flush, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	c.flush = flush

	return c
}

func (c *Client) Send(ev *Event) {
	io.Copy(c.write, ev)
	c.flush.Flush()
}

func (c *Client) run() {

}
