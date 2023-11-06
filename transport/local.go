package transport

import wamp "github.com/wamp3hub/wamp3go"

type QEvent chan wamp.Event

type localTransport struct {
	tq QEvent
	rq QEvent
}

func NewDuplexLocalTransport(qSize int) (*localTransport, *localTransport) {
	left := make(QEvent, qSize)
	right := make(QEvent, qSize)
	return &localTransport{left, right}, &localTransport{right, left}
}

func (transport *localTransport) Close() error {
	close(transport.tq)
	close(transport.rq)
	return nil
}

func (transport *localTransport) Write(event wamp.Event) error {
	transport.tq <- event
	return nil
}

func (transport *localTransport) Read() (wamp.Event, error) {
	event := <-transport.rq
	return event, nil
}
