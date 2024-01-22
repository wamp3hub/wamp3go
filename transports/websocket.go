package wampTransports

import (
	"fmt"
	"log/slog"

	"github.com/gorilla/websocket"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampShared "github.com/wamp3hub/wamp3go/shared"
	wampInterview "github.com/wamp3hub/wamp3go/transports/interview"
)

type WSTransport struct {
	Address              string
	Serializer           wamp.Serializer
	Connection           *websocket.Conn
}

func (transport *WSTransport) Connect() (e error) {
	transport.Connection, _, e = websocket.DefaultDialer.Dial(transport.Address, nil)
	return e
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
	return nil, wamp.ErrorConnectionLost
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

	logger := slog.New(joinOptions.LoggingHandler)
	joinOptionsLogData := slog.Group(
		"joinOptions",
		"address", joinOptions.Address,
		"secure", joinOptions.Secure,
		"serializer", joinOptions.Serializer.Code(),
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

	interviewLogData := slog.Group("interview", "peerID", payload.YourID, "routerID", payload.RouterID, "ticket", payload.Ticket)
	logger.Debug("interview has been completed", joinOptionsLogData, interviewLogData)

	protocol := "ws"
	if joinOptions.Secure {
		protocol = "wss"
	}
	wsAddress := fmt.Sprintf(
		"%s://%s/wamp/v1/websocket?ticket=%s&serializerCode=%s",
		protocol, joinOptions.Address, payload.Ticket, joinOptions.Serializer.Code(),
	)
	transport := WSTransport{Address: wsAddress, Serializer: joinOptions.Serializer}
	e = transport.Connect()
	if e == nil {
		peer := wamp.SpawnPeer(payload.YourID, &transport, joinOptions.ReconnectionStrategy, logger)
		session := wamp.NewSession(peer, logger)
		logger.Debug("successful joined", joinOptionsLogData, interviewLogData)
		return session, nil
	}

	logger.Error("during connect", "error", e, joinOptionsLogData, interviewLogData)
	return nil, e
}
