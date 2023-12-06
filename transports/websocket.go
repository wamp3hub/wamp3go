package wampTransports

import (
	"fmt"
	"log/slog"
	"os"

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
		return transport.Serializer.Decode(rawMessage)
	}
	return nil, wamp.ErrorConnectionLost
}

func WebsocketConnect(
	address string,
	serializer wamp.Serializer,
	logger *slog.Logger,
) (wamp.Transport, error) {
	logData := slog.Group("websocket", "address", address, "serializer", serializer.Code())
	logger.Debug("trying to connect", logData)
	connection, response, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		transport := WSTransport(serializer, connection)
		return transport, nil
	}

	logger.Debug("during connect", "response.Status", response.Status, "error", e, logData)
	return nil, e
}

func WebsocketJoin(
	address string,
	secure bool,
	serializer wamp.Serializer,
	credentials any,
) (*wamp.Session, error) {
	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug},
	)
	logger := slog.New(handler)

	payload, e := wampInterview.HTTP2Interview(
		address,
		secure,
		&wampInterview.Payload{Credentials: credentials},
	)
	if e == nil {
		protocol := "ws"
		if secure {
			protocol = "wss"
		}
		wsAddress := fmt.Sprintf(
			"%s://%s/wamp/v1/websocket?ticket=%s&serializerCode=%s",
			protocol, address, payload.Ticket, serializer.Code(),
		)
		transport, e := WebsocketConnect(wsAddress, serializer, logger)
		if e == nil {
			peer := wamp.SpawnPeer(payload.YourID, transport, logger)
			session := wamp.NewSession(peer, logger)
			logger.Debug("joined", "peer.ID", peer.ID, "address", address)
			return session, nil
		}
		return nil, e
	}
	return nil, e
}
