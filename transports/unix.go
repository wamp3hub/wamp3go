package wampTransports

import (
	"net"

	"github.com/wamp3hub/wamp3go"
)

type unixTransport struct {
	Serializer wamp.Serializer
	Connection net.Conn
}

func UnixTransport(
	serializer wamp.Serializer,
	connection net.Conn,
) wamp.Transport {
	return &unixTransport{serializer, connection}
}

func (transport *unixTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func (transport *unixTransport) Write(event wamp.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		_, e = transport.Connection.Write(rawMessage)
	}
	return e
}

func (transport *unixTransport) Read() (wamp.Event, error) {
	buffer := make([]byte, 1024)
	n, e := transport.Connection.Read(buffer)
	if e == nil {
		rawMessage := buffer[0:n]
		event, e := transport.Serializer.Decode(rawMessage)
		if e == nil {
			return event, nil
		}
	}
	return nil, e
}

func UnixConnect(
	address string,
	serializer wamp.Serializer,
) (wamp.Transport, error) {
	connection, e := net.Dial("unix", address)
	if e == nil {
		transport := UnixTransport(serializer, connection)
		return transport, nil
	}
	return nil, e
}

func UnixJoin(
	address string,
	serializer wamp.Serializer,
) (*wamp.Session, error) {
	transport, e := UnixConnect(address, serializer)
	if e == nil {
		peer := wamp.SpawnPeer("", transport)
		session := wamp.NewSession(peer)
		return session, nil
	}
	return nil, e
}
