package wampTransports

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net"
	"time"

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

func (transport *unixTransport) Read() (event wamp.Event, e error) {
	rawMessage, e := transport.ReadRaw()
	if e == nil {
		return transport.Serializer.Decode(rawMessage)
	}
	return nil, ErrorBadConnection
}

func (transport *unixTransport) initialize() (string, string, error) {
	serverMessage := new(UnixServerMessage)
	rawServerMessage, e := transport.ReadRaw()
	if e == nil {
		e = json.Unmarshal(rawServerMessage, serverMessage)
		if e == nil {
			clientMessage := UnixClientMessage{transport.Serializer.Code()}
			rawClientMessage, _ := json.Marshal(clientMessage)
			e = transport.WriteRaw(rawClientMessage)
			if e == nil {
				return serverMessage.RouterID, serverMessage.YourID, nil
			}
		}
	}
	return "", "", e
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
	timeout time.Duration,
	serializer wamp.Serializer,
) (*unixTransport, error) {
	if timeout == 0 {
		timeout = time.Minute
	}
	connection, e := net.DialTimeout("unix", path, timeout)
	if e == nil {
		transport := UnixTransport(serializer, connection)
		return transport, nil
	}
	return nil, e
}

type UnixJoinOptions struct {
	Path           string
	DialTimeout    time.Duration
	Serializer     wamp.Serializer
	LoggingHandler slog.Handler
}

func UnixJoin(
	joinOptions *UnixJoinOptions,
) (*wamp.Session, error) {
	logger := slog.New(joinOptions.LoggingHandler)
	logData := slog.Group(
		"JoinOptions",
		"Path", joinOptions.Path,
		"Serializer", joinOptions.Serializer.Code(),
	)
	logger.Debug("trying to join", logData)

	transport, e := UnixConnect(joinOptions.Path, joinOptions.DialTimeout, joinOptions.Serializer)
	if e != nil {
		logger.Error("failed to connect unix server", "error", e, logData)
		return nil, e
	}

	routerID, peerID, e := transport.initialize()
	if e == nil {
		peer := wamp.SpawnPeer(peerID, transport, logger)
		session := wamp.NewSession(peer, logger)
		logger.Debug("successfully joined", "routerID", routerID, "peerID", peerID, logData)
		return session, nil
	}
	logger.Error("failed to initialize unix transport", "error", e, logData)
	return nil, e
}
