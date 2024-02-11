package main

import (
	"log/slog"

	wamp "github.com/wamp3hub/wamp3go"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	logger := slog.Default()

	type LoginPayload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	session, e := wampTransports.WebsocketJoin(
		"0.0.0.0:8800",
		"customer",
		&wampTransports.WebsocketJoinOptions{
			Credentials: &LoginPayload{"test", "secret"},
		},
	)
	if e == nil {
		logger.Info("WAMP Join Success", "sessionID", session.ID(), "role", session.Role())
	} else {
		logger.Error("during WAMP join", "error", e)
		panic("WAMP Join Error")
	}

	wamp.Publish(session, &wamp.PublishFeatures{URI: "net.example"}, "Hello, WAMP!")
	wamp.Publish(session, &wamp.PublishFeatures{URI: "net.example"}, "How are you?")

	wamp.Leave(session, "done")
}
