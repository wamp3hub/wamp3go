package wampTransports

import (
	"bufio"
	"encoding/json"
	"log"
	"log/slog"
	"net"
	"os"

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
		return transport.Serializer.Decode(rawMessage)
	}
	return nil, wamp.ErrorConnectionLost
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
	logger *slog.Logger,
) (wamp.Transport, string, error) {
	logData := slog.Group("unix", "path", path, "serializer", serializer.Code())
	logger.Debug("trying to connect", logData)
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
	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug},
	)
	logger := slog.New(handler)

	transport, peerID, e := UnixConnect(path, serializer, logger)
	if e == nil {
		peer := wamp.SpawnPeer(peerID, transport, logger)
		session := wamp.NewSession(peer, logger)
		log.Printf("[unix] peer.ID=%s joined to %s", peer.ID, path)
		return session, nil
	}
	return nil, e
}
