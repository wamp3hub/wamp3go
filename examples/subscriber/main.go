package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	subscription, e := wamp.Subscribe(
		session,
		"net.example",
		&wamp.SubscribeOptions{},
		func(message string, publishEvent wamp.PublishEvent) {
			logger.Info("new", "message", message)
		},
	)
	if e == nil {
		logger.Info("subscribe success", "subscriptionID", subscription.ID)
	} else {
		logger.Error("during subscribe", "error", e)
		panic("subscribe error")
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

	wamp.Leave(session, "done")
}
