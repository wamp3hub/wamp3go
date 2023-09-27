package transport

import (
	"log"

	client "wamp3go"
	clientJoin "wamp3go/transport/join"

	"github.com/gorilla/websocket"
)

type WSTransport struct {
	Serializer client.Serializer
	Connection *websocket.Conn
}

func (transport *WSTransport) Send(event client.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
	} else {
		log.Printf("[WSTransport] EncodeMessage: %s", e)
	}
	return e
}

func (transport *WSTransport) Receive(q client.QEvent) {
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
			log.Printf("[WSTransport] DecodeMessage: %s", e)
		}
	}
	close(q)
}

func (transport *WSTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func connectWebsocket(
	address string,
	token string,
	serializer client.Serializer,
) (*WSTransport, error) {
	wsAddress := "ws://" + address + "/wamp3/websocket?token=" + token
	log.Printf("websocket dial %s", wsAddress)
	connection, response, e := websocket.DefaultDialer.Dial(wsAddress, nil)
	if e == nil {
		return &WSTransport{serializer, connection}, nil
	}
	log.Printf("%s websocket connect %s", response.Status, e)
	return nil, e
}

func WebsocketJoin(
	address string,
	serializer client.Serializer,
) (*client.Session, error) {
	joinPayload := clientJoin.JoinPayload{serializer.Code()}
	response, e := clientJoin.HTTP2Join(address, &joinPayload)
	if e == nil {
		transport, e := connectWebsocket(address, response.Token, serializer)
		if e == nil {
			peer := client.NewPeer(response.PeerID, transport)
			go peer.Consume()
			session := client.NewSession(peer)
			return session, nil
		}
	}
	return nil, e
}
