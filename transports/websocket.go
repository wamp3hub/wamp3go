package wampTransports

import (
	"log"

	"github.com/gorilla/websocket"

	wamp "github.com/wamp3hub/wamp3go"
	wampInterview "github.com/wamp3hub/wamp3go/transports/interview"
)

type wsTransport struct {
	Serializer wamp.Serializer
	Connection *websocket.Conn
}

func WSTransport(
	serializer wamp.Serializer,
	connection *websocket.Conn,
) *wsTransport {
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
) (wamp.Transport, error) {
	log.Printf("[http2websocket] dial %s", address)
	connection, response, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		transport := WSTransport(serializer, connection)
		return transport, nil
	}
	log.Printf("[http2websocket] connect statusCode=%s error=%s", response.Status, e)
	return nil, e
}

func WebsocketJoin(
	address string,
	secure bool,
	serializer wamp.Serializer,
	credentials any,
) (*wamp.Session, error) {
	log.Printf("[http2websocket] trying to join %s", address)
	payload, e := wampInterview.HTTP2Interview(
		address,
		secure,
		&wampInterview.Payload{Credentials: credentials},
	)
	if e == nil {
		wsAddress := "ws://" + address + "/wamp/v1/websocket?ticket=" + payload.Ticket
		transport, e := WebsocketConnect(wsAddress, serializer)
		if e == nil {
			peer := wamp.SpawnPeer(payload.YourID, transport)
			session := wamp.NewSession(peer)
			log.Printf("[http2websocket] peer.ID=%s joined", peer.ID)
			return session, nil
		}
		return nil, e
	}
	return nil, e
}
