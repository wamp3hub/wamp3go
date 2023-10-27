package transport

import (
	"log"

	"github.com/gorilla/websocket"

	wamp "github.com/wamp3hub/wamp3go"
	wampInterview "github.com/wamp3hub/wamp3go/transport/interview"
)

type wsTransport struct {
	Serializer wamp.Serializer
	Connection *websocket.Conn
}

func WSTransport(
	serializer wamp.Serializer,
	connection *websocket.Conn,
) wamp.Transport {
	return &wsTransport{serializer, connection}
}

func (transport *wsTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func (transport *wsTransport) Write(event wamp.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
	}
	return e
}

func (transport *wsTransport) Read() (wamp.Event, error) {
	_, rawMessage, e := transport.Connection.ReadMessage()
	if e == nil {
		event, e := transport.Serializer.Decode(rawMessage)
		if e == nil {
			return event, nil
		}
	}
	return nil, e
}

func WebsocketConnect(
	address string,
	serializer wamp.Serializer,
) (string, wamp.Transport, error) {
	log.Printf("[websocket] dial %s", address)
	connection, response, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		routerID := response.Header.Get("X-WAMP-RouterID")
		transport := WSTransport(serializer, connection)
		return routerID, transport, nil
	}
	log.Printf("[websocket] %s connect %s", response.Status, e)
	return "", nil, e
}

func WebsocketJoin(
	address string,
	serializer wamp.Serializer,
	credentials any,
) (*wamp.Session, error) {
	payload, e := wampInterview.HTTP2Interview(
		address,
		&wampInterview.Payload{Credentials: credentials},
	)
	if e == nil {
		wsAddress := "ws://" + address + "/wamp3/websocket?ticket=" + payload.Ticket
		_, transport, e := WebsocketConnect(wsAddress, serializer)
		if e == nil {
			peer := wamp.SpawnPeer(payload.YourID, transport)
			session := wamp.NewSession(peer)
			return session, nil
		}
	}
	return nil, e
}
