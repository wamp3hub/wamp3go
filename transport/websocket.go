package transport

import (
	"log"

	"github.com/gorilla/websocket"

	client "github.com/wamp3hub/wamp3go"
	interview "github.com/wamp3hub/wamp3go/transport/interview"
)

type wsTransport struct {
	Serializer client.Serializer
	Connection *websocket.Conn
}

func WSTransport(
	serializer client.Serializer,
	connection *websocket.Conn,
) client.Transport {
	return &wsTransport{serializer, connection}
}

func (transport *wsTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func (transport *wsTransport) Write(event client.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
	}
	return e
}

func (transport *wsTransport) Read() (client.Event, error) {
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
	serializer client.Serializer,
) (client.Transport, error) {
	log.Printf("[websocket] dial %s", address)
	connection, response, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		return WSTransport(serializer, connection), nil
	}
	log.Printf("[websocket] %s connect %s", response.Status, e)
	return nil, e
}

func WebsocketJoin(
	address string,
	serializer client.Serializer,
	credentials any,
) (*client.Session, error) {
	payload, e := interview.HTTP2Interview(address, &interview.Payload{Credentials: credentials})
	if e == nil {
		wsAddress := "ws://" + address + "/wamp3/websocket?ticket=" + payload.Ticket
		transport, e := WebsocketConnect(wsAddress, serializer)
		if e == nil {
			peer := client.SpawnPeer(payload.YourID, transport)
			session := client.NewSession(peer)
			return session, nil
		}
	}
	return nil, e
}
