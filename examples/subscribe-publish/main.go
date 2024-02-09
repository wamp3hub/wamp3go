package main

import (
	"log/slog"
	"sync"

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

	wg := new(sync.WaitGroup)
	wg.Add(2)

	subscription, e := wamp.Subscribe(
		session,
		"net.example",
		&wamp.SubscribeOptions{},
		func(message string, publishEvent wamp.PublishEvent) {
			logger.Info("new", "message", message)
			wg.Done()
		},
	)
	if e == nil {
		logger.Info("subscribe success", "subscriptionID", subscription.ID)
	} else {
		logger.Error("during subscribe", "error", e)
		panic("subscribe error")
	}

	wamp.Publish(session, &wamp.PublishFeatures{URI: "net.example"}, "Hello, WAMP!")
	wamp.Publish(session, &wamp.PublishFeatures{URI: "net.example"}, "How are you?")

	wg.Wait()

	wamp.Leave(session, "done")
}
