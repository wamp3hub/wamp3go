package transport

import (
	"log"

	client "wamp3go"

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
	}
	return e
}

func (transport *WSTransport) Receive(q client.QEvent) {
	for {
		WSMessageType, rawMessage, e := transport.Connection.ReadMessage()
		if e != nil {
			log.Printf("WSTransport.receive messageType=%d: %s", WSMessageType, e)
			break
		}
		event, e := transport.Serializer.Decode(rawMessage)
		if e == nil {
			q <- event
		} else {
			log.Printf("WSTransport.receive InvalidMessage: %s", e)
		}
	}
	close(q)
}

func (transport *WSTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func connectWebsocket(url string, serializer client.Serializer) (client.Transport, error) {
	log.Printf("websocket dial %s", url)
	connection, response, e := websocket.DefaultDialer.Dial(url, nil)
	if e != nil {
		log.Printf("%s websocket connect %s", response.Status, e)
		return nil, e
	}
	return &WSTransport{serializer, connection}, nil
}

func WebsocketJoin(
	address string,
	serializer client.Serializer,
) (*client.Session, error) {
	joinPayload := client.JoinPayload{"", serializer.Code()}
	response, e := client.Join(address, &joinPayload)
	if e == nil {
		wsAddress := "ws://" + address + "/wamp3/websocket?token=" + response.Token
		transport, e := connectWebsocket(wsAddress, serializer)
		if e == nil {
			peer := client.NewPeer(response.PeerID, transport)
			session := client.NewSession(peer)
			client.Initialize(session)
			return session, nil
		}
	}
	return nil, e
}
