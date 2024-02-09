package main

import (
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	wamp "github.com/wamp3hub/wamp3go"
	wampInterview "github.com/wamp3hub/wamp3go/interview"
	wampTransports "github.com/wamp3hub/wamp3go/transports"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func authenticate(payload *LoginPayload) error {
	availableUsers := map[string]string{
		"main": "passw0rd!",
		"test": "secret",
	}
	expectedPassword, ok := availableUsers[payload.Username]
	if !ok || payload.Password != expectedPassword {
		return errors.New("user not found")
	}
	return nil
}

func main() {
	logger := slog.Default()

	session, e := wampTransports.WebsocketJoin(
		"0.0.0.0:8800",
		"interviewer",
		&wampTransports.WebsocketJoinOptions{
			Credentials: &LoginPayload{"main", "passw0rd!"},
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
		"wamp.authenticate",
		&wamp.RegisterOptions{},
		func(resume wampInterview.Resume[*LoginPayload], callEvent wamp.CallEvent) (*wampInterview.Offer, error) {
			if resume.Role == "guest" {
				offer := wampInterview.Offer{
					RegistrationsLimit: 0,
					SubscriptionsLimit: 0,
				}
				return &offer, nil
			}

			e := authenticate(resume.Credentials)
			if e == nil {
				logger.Info("customer successfully authenticated", "username", resume.Credentials.Username)
				offer := wampInterview.Offer{
					RegistrationsLimit: 100,
					SubscriptionsLimit: 100,
				}
				return &offer, nil
			}
			logger.Error("during authenticate", "error", e, "username", resume.Credentials.Username)
			return nil, e
		},
	)
	if e == nil {
		logger.Debug("registration successfully created", "registrationID", registration.ID)
	} else {
		logger.Error("during register procedure", "error", e)
		panic("register procedure error")
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

	wamp.Leave(session, "exit signal")
}
