package wampTransports

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gorilla/websocket"

	wamp "github.com/wamp3hub/wamp3go"
	wampInterview "github.com/wamp3hub/wamp3go/interview"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampShared "github.com/wamp3hub/wamp3go/shared"
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
	if websocket.IsCloseError(e, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		return nil, wamp.ErrorConnectionClosed
	}
	return nil, ErrorBadConnection
}

func dialWebsocket(
	address string,
	serializer wamp.Serializer,
) (wamp.Transport, error) {
	connection, _, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		instance := WSTransport{address, serializer, connection}
		return &instance, nil
	}
	return nil, e
}

func WebsocketConnect(
	address string,
	serializer wamp.Serializer,
	strategy wampShared.RetryStrategy,
	logger *slog.Logger,
) (wamp.Transport, error) {
	instance, e := MakeReconnectable(
		func() (wamp.Transport, error) {
			return dialWebsocket(address, serializer)
		},
		strategy,
		logger,
	)
	return instance, e
}

type WebsocketJoinOptions struct {
	Secure               bool
	Serializer           wamp.Serializer
	Credentials          any
	LoggingHandler       slog.Handler
	ReconnectionStrategy wampShared.RetryStrategy
}

func WebsocketJoin(
	address string,
	role string,
	joinOptions *WebsocketJoinOptions,
) (*wamp.Session, error) {
	if len(role) == 0 {
		role = "guest"
	}
	if joinOptions == nil {
		joinOptions = new(WebsocketJoinOptions)
	}
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
		"options",
		"role", role,
		"address", address,
		"secure", joinOptions.Secure,
		"serializer", joinOptions.Serializer.Code(),
	)
	logger.Debug("trying to join", joinOptionsLogData)

	interviewResult, e := wampInterview.HTTP2Interview(
		address,
		joinOptions.Secure,
		&wampInterview.Resume[any]{
			Role:        role,
			Credentials: joinOptions.Credentials,
		},
	)
	if e != nil {
		logger.Error("during interview", "error", e, joinOptionsLogData)
		return nil, e
	}

	interviewLogData := slog.Group(
		"interview",
		"peerID", interviewResult.YourID,
		"routerID", interviewResult.RouterID,
	)
	logger.Debug("interview successfully", joinOptionsLogData, interviewLogData)

	protocol := "ws"
	if joinOptions.Secure {
		protocol = "wss"
	}
	wsAddress := fmt.Sprintf(
		"%s://%s/wamp/v1/websocket?ticket=%s&serializerCode=%s",
		protocol, address, interviewResult.Ticket, joinOptions.Serializer.Code(),
	)
	transport, e := WebsocketConnect(
		wsAddress, joinOptions.Serializer, joinOptions.ReconnectionStrategy, logger,
	)
	if e != nil {
		logger.Error("during connect", "error", e, joinOptionsLogData, interviewLogData)
		return nil, e
	}

	peerDetails := wamp.PeerDetails{
		ID:    interviewResult.YourID,
		Role:  role,
		Offer: interviewResult.Offer,
	}
	peer := wamp.SpawnPeer(&peerDetails, transport, logger)
	session := wamp.NewSession(peer, logger)
	logger.Debug("successfully joined", joinOptionsLogData, interviewLogData)
	return session, nil
}
