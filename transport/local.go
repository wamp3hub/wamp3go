package transport

import client "github.com/wamp3hub/wamp3go"

type localTransport struct {
	tq client.QEvent
	rq client.QEvent
}

func NewDuplexLocalTransport() (*localTransport, *localTransport) {
	alpha := make(client.QEvent)
	beta := make(client.QEvent)
	return &localTransport{alpha, beta}, &localTransport{beta, alpha}
}

func (transport *localTransport) Send(event client.Event) error {
	transport.tq <- event
	return nil
}

func (transport *localTransport) Receive(q client.QEvent) {
	for event := range transport.rq {
		q <- event
	}
	close(q)
}

func (transport *localTransport) Close() error {
	close(transport.tq)
	close(transport.rq)
	return nil
}
