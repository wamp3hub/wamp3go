package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializers "github.com/wamp3hub/wamp3go/serializers"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

func main() {
	type LoginPayload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	session, e := wampTransports.WebsocketJoin(
		&wampTransports.WebsocketJoinOptions{
			Secure:      false,
			Address:     "0.0.0.0:8800",
			Serializer:  wampSerializers.DefaultSerializer,
			Credentials: &LoginPayload{"test", "test"},
			LoggingHandler: slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo},
			),
		},
	)
	if e == nil {
		fmt.Printf("WAMP Join Success\n")
	} else {
		panic("WAMP Join Error")
	}

	wg := new(sync.WaitGroup)

	registration, e := wamp.Register(
		session,
		"net.example.greeting",
		&wamp.RegisterOptions{Description: "greeting"},
		func(name string, callEvent wamp.CallEvent) (string, error) {
			wg.Done()
			if len(name) == 0 {
				return "", errors.New("InvalidName")
			}
			result := "Hello, " + name + "!"
			return result, nil
		},
	)
	if e == nil {
		fmt.Printf("register success ID=%s\n", registration.ID)
	} else {
		panic("register error")
	}

	type ProjectPayload struct {
		Name string `json:"name"`
	}

	type CreateUserPayload struct {
		Age      int              `json:"age"`
		Name     string           `json:"name" jsonschema:"default=lucy"`
		Password string           `json:"password"`
		Roles    []string         `json:"roles"`
		Projects []ProjectPayload `json:"projects"`
	}

	registration, e = wamp.Register(
		session,
		"net.example.user.create",
		&wamp.RegisterOptions{Description: "creates new user"},
		func(payload CreateUserPayload, callEvent wamp.CallEvent) (CreateUserPayload, error) {
			wg.Done()
			return payload, nil
		},
	)
	if e == nil {
		fmt.Printf("register success ID=%s\n", registration.ID)
	} else {
		panic("register error")
	}

	wg.Add(100)
	wg.Wait()
}
