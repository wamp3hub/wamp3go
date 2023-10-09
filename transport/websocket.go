package transport

import (
	"log"

	client "github.com/wamp3hub/wamp3go"
	interview "github.com/wamp3hub/wamp3go/transport/interview"

	"github.com/gorilla/websocket"
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

func (transport *wsTransport) Send(event client.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
	} else {
		log.Printf("[WSTransport] EncodeMessage: %s", e)
	}
	return e
}

func (transport *wsTransport) Receive(q client.QEvent) {
	for {
		WSMessageType, rawMessage, e := transport.Connection.ReadMessage()
		if e != nil {
			log.Printf("[WSTransport] %s (messageType=%d)", e, WSMessageType)
			break
		}
		event, e := transport.Serializer.Decode(rawMessage)
		if e == nil {
			q <- event
		} else {
			log.Printf("[WSTransport] e=%s raw=%s", e, rawMessage)
		}
	}
	close(q)
}

func (transport *wsTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func WebsocketConnect(
	address string,
	serializer client.Serializer,
) (string, client.Transport, error) {
	log.Printf("[websocket] dial %s", address)
	connection, response, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		routerID := response.Header.Get("X-WAMP-RouterID")
		return routerID, WSTransport(serializer, connection), nil
	}
	log.Printf("[websocket] %s connect %s", response.Status, e)
	return "", nil, e
}

func WebsocketJoin(
	address string,
	serializer client.Serializer,
	credentials any,
) (*client.Session, error) {
	payload, e := interview.HTTP2Interview(address, &interview.Payload{credentials})
	if e == nil {
		wsAddress := "ws://" + address + "/wamp3/websocket?token=" + payload.Token
		_, transport, e := WebsocketConnect(wsAddress, serializer)
		if e == nil {
			peer := client.NewPeer(payload.PeerID, transport)
			go peer.Consume()
			session := client.NewSession(peer)
			return session, nil
		}
	}
	return nil, e
}
