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

	pendingResponse := wamp.Call[string](
		session,
		&wamp.CallFeatures{
			URI:          "net.example.greeting",
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
