package wampTransports

import (
	"fmt"
	"log/slog"

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
) (*wsTransport, error) {
	connection, _, e := websocket.DefaultDialer.Dial(address, nil)
	if e == nil {
		transport := WSTransport(serializer, connection)
		return transport, nil
	}
	return nil, e
}

type WebsocketJoinOptions struct {
	Secure         bool
	Address        string
	Serializer     wamp.Serializer
	Credentials    any
	LoggingHandler slog.Handler
}

func WebsocketJoin(
	joinOptions *WebsocketJoinOptions,
) (*wamp.Session, error) {
	logger := slog.New(joinOptions.LoggingHandler)
	joinOptionsLogData := slog.Group("joinOptions", "address", joinOptions.Address, "secure", joinOptions.Secure, "serializer", joinOptions.Serializer.Code())
	logger.Debug("trying to join", joinOptionsLogData)

	payload, e := wampInterview.HTTP2Interview(
		joinOptions.Address,
		joinOptions.Secure,
		&wampInterview.Payload{Credentials: joinOptions.Credentials},
	)
	if e != nil {
		logger.Error("interview failed", "error", e, joinOptionsLogData)
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
	transport, e := WebsocketConnect(wsAddress, joinOptions.Serializer)
	if e == nil {
		peer := wamp.SpawnPeer(payload.YourID, transport, logger)
		session := wamp.NewSession(peer, logger)
		logger.Debug("successful joined", joinOptionsLogData, interviewLogData)
		return session, nil
	}

	logger.Error("join failed", "error", e, joinOptionsLogData, interviewLogData)
	return nil, e
}
