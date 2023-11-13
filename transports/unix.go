package wampTransports

import (
	"bufio"
	"encoding/json"
	"log"
	"net"

	wamp "github.com/wamp3hub/wamp3go"
)

type unixTransport struct {
	Serializer wamp.Serializer
	Connection net.Conn
	buffer     *bufio.Reader
}

func UnixTransport(
	serializer wamp.Serializer,
	connection net.Conn,
) *unixTransport {
	return &unixTransport{
		serializer,
		connection,
		bufio.NewReader(connection),
	}
}

func (transport *unixTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func (transport *unixTransport) WriteRaw(data []byte) error {
	data = append(data, byte('\n'))
	_, e := transport.Connection.Write(data)
	return e
}

func (transport *unixTransport) WriteJSON(payload any) error {
	rawMessage, e := json.Marshal(payload)
	if e == nil {
		e = transport.WriteRaw(rawMessage)
	}
	return e
}

func (transport *unixTransport) Write(event wamp.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.WriteRaw(rawMessage)
	}
	return e
}

func (transport *unixTransport) ReadRaw() ([]byte, error) {
	rawMessage, _, e := transport.buffer.ReadLine()
	return rawMessage, e
}

func (transport *unixTransport) ReadJSON(payload any) error {
	rawMessage, e := transport.ReadRaw()
	if e == nil {
		e = json.Unmarshal(rawMessage, payload)
	}
	return e
}

func (transport *unixTransport) Read() (event wamp.Event, e error) {
	rawMessage, e := transport.ReadRaw()
	if e == nil {
		event, e = transport.Serializer.Decode(rawMessage)
		if e == nil {
			return event, nil
		}
	}
	return nil, e
}

type UnixServerMessage struct {
	RouterID string `json:"routerID"`
	YourID   string `json:"yourID"`
}

type UnixClientMessage struct {
	SerializerCode string `json:"serializerCode"`
}

func UnixConnect(
	path string,
	serializer wamp.Serializer,
) (wamp.Transport, string, error) {
	log.Printf("[unix] dial %s", path)
	connection, e := net.Dial("unix", path)
	if e == nil {
		transport := UnixTransport(serializer, connection)
		serverMessage := new(UnixServerMessage)
		e = transport.ReadJSON(serverMessage)
		if e == nil {
			clientMessage := UnixClientMessage{serializer.Code()}
			e = transport.WriteJSON(clientMessage)
			if e == nil {
				return transport, serverMessage.YourID, nil
			}
		}
	}
	return nil, "", e
}

func UnixJoin(
	path string,
	serializer wamp.Serializer,
) (*wamp.Session, error) {
	log.Printf("[unix] trying to join %s", path)
	transport, peerID, e := UnixConnect(path, serializer)
	if e == nil {
		peer := wamp.SpawnPeer(peerID, transport)
		session := wamp.NewSession(peer)
		log.Printf("[unix] peer.ID=%s joined to %s", peer.ID, path)
		return session, nil
	}
	return nil, e
}
