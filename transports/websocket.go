package wampTransports

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gorilla/websocket"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampShared "github.com/wamp3hub/wamp3go/shared"
	wampInterview "github.com/wamp3hub/wamp3go/transports/interview"
)

type WSTransport struct {
	Address    string
	Serializer wamp.Serializer
	Connection *websocket.Conn
}

func (transport *WSTransport) Close() error {
	e := transport.Connection.Close()
	return e
}

func (transport *WSTransport) Write(event wamp.Event) error {
	rawMessage, e := transport.Serializer.Encode(event)
	if e == nil {
		e = transport.Connection.WriteMessage(websocket.TextMessage, rawMessage)
	}
	return e
}

func (transport *WSTransport) Read() (wamp.Event, error) {
	_, rawMessage, e := transport.Connection.ReadMessage()
	if e == nil {
		return transport.Serializer.Decode(rawMessage)
	}
	if websocket.IsCloseError(e, websocket.CloseNormalClosure) {
		return nil, wamp.ErrorConnectionClosed
	}
	return nil, ErrorBadConnection
}

func WebsocketConnect(
	address string,
	serializer wamp.Serializer,
	strategy wampShared.RetryStrategy,
	logger *slog.Logger,
) (wamp.Transport, error) {
	connect := func() (wamp.Transport, error) {
		connection, _, e := websocket.DefaultDialer.Dial(address, nil)
		if e == nil {
			instance := WSTransport{address, serializer, connection}
			return &instance, nil
		}
		return nil, e
	}

	transport, e := connect()
	if e == nil {
		instance := NewReconnectableTransport(
			transport,
			strategy,
			connect,
			logger,
		)
		return instance, nil
	}
	return nil, e
}

type WebsocketJoinOptions struct {
	Secure               bool
	Address              string
	Serializer           wamp.Serializer
	Credentials          any
	LoggingHandler       slog.Handler
	ReconnectionStrategy wampShared.RetryStrategy
}

func WebsocketJoin(
	joinOptions *WebsocketJoinOptions,
) (*wamp.Session, error) {
	if joinOptions.Serializer == nil {
		joinOptions.Serializer = wampSerializers.DefaultSerializer
	}
	if joinOptions.ReconnectionStrategy == nil {
		joinOptions.ReconnectionStrategy = wampShared.DefaultRetryStrategy
	}
	if joinOptions.LoggingHandler == nil {
		joinOptions.LoggingHandler = slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo},
		)
	}

	logger := slog.New(joinOptions.LoggingHandler)
	joinOptionsLogData := slog.Group(
		"joinOptions",
		"address", joinOptions.Address,
		"secure", joinOptions.Secure,
	)
	logger.Debug("trying to join", joinOptionsLogData)

	payload, e := wampInterview.HTTP2Interview(
		joinOptions.Address,
		joinOptions.Secure,
		&wampInterview.Payload{Credentials: joinOptions.Credentials},
	)
	if e != nil {
		logger.Error("during interview", "error", e, joinOptionsLogData)
		return nil, e
	}

	interviewLogData := slog.Group(
		"interview",
		"peerID", payload.YourID,
		"routerID", payload.RouterID,
		"ticket", payload.Ticket,
	)
	logger.Debug("interview has been completed", joinOptionsLogData, interviewLogData)

	protocol := "ws"
	if joinOptions.Secure {
		protocol = "wss"
	}
	wsAddress := fmt.Sprintf(
		"%s://%s/wamp/v1/websocket?ticket=%s&serializerCode=%s",
		protocol, joinOptions.Address, payload.Ticket, joinOptions.Serializer.Code(),
	)
	transport, e := WebsocketConnect(
		wsAddress, joinOptions.Serializer, joinOptions.ReconnectionStrategy, logger,
	)
	if e != nil {
		logger.Error("during connect", "error", e, joinOptionsLogData, interviewLogData)
		return nil, e
	}

	peer := wamp.SpawnPeer(payload.YourID, transport, logger)
	session := wamp.NewSession(peer, logger)
	logger.Debug("successfully joined", joinOptionsLogData, interviewLogData)
	return session, nil
}
