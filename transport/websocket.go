package transport

import (
	"log"
	"sync"

	client "github.com/wamp3hub/wamp3go"
	interview "github.com/wamp3hub/wamp3go/transport/interview"

	"github.com/gorilla/websocket"
)

type wsTransport struct {
	Serializer client.Serializer
	Connection *websocket.Conn
	writeMutex *sync.Mutex
}

func WSTransport(
	serializer client.Serializer,
	connection *websocket.Conn,
) client.Transport {
	return &wsTransport{serializer, connection, new(sync.Mutex)}
}

func (transport *wsTransport) Send(event client.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		transport.writeMutex.Lock()
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
		transport.writeMutex.Unlock()
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

func connectWebsocket(
	address string,
	token string,
	serializer client.Serializer,
) (client.Transport, error) {
	wsAddress := "ws://" + address + "/wamp3/websocket?token=" + token
	log.Printf("[websocket] dial %s", wsAddress)
	connection, response, e := websocket.DefaultDialer.Dial(wsAddress, nil)
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
	payload, e := interview.HTTP2Interview(
		address, &interview.Payload{credentials},
	)
	if e == nil {
		transport, e := connectWebsocket(address, payload.Token, serializer)
		if e == nil {
			peer := client.NewPeer(payload.PeerID, transport)
			go peer.Consume()
			session := client.NewSession(peer)
			return session, nil
		}
	}
	return nil, e
}
