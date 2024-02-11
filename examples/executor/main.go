package main

import (
	"errors"
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

	registration, e := wamp.Register(
		session,
		"net.example.greeting",
		&wamp.RegisterOptions{
			IncludeRoles: []string{"customer", "guest"},
		},
		func(name string, callEvent wamp.CallEvent) (string, error) {
			if len(name) == 0 {
				return "", errors.New("InvalidName")
			}
			result := "Hello, " + name + "!"
			return result, nil
		},
	)
	if e == nil {
		logger.Info("register success", "registrationID", registration.ID)
	} else {
		logger.Error("during register", "error", e)
		panic("register error")
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

	wamp.Leave(session, "done")
}
