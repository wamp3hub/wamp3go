package main

import (
	"errors"
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

	pendingResponse := wamp.Call[string](
		session,
		&wamp.CallFeatures{
			URI: "net.example.greeting",
			IncludeRoles: []string{"customer"},
		},
		"WAMP",
	)
	_, result, e := pendingResponse.Await()
	if e == nil {
		logger.Info("call(example.greeting)", "result", result)
	} else {
		logger.Error("call(example.greeting)", "error", e)
	}

	wamp.Leave(session, "done")
}
