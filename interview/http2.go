package wampInterview

import (
	"fmt"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

type Resume[T any] struct {
	Role        string `json:"role"`
	Credentials T      `json:"credentials"`
}

type Offer struct {
	RegistrationsLimit uint32 `json:"registrationsLimit"`
	SubscriptionsLimit uint32 `json:"subscriptionsLimit"`
	TicketLifeTime     uint64 `json:"ticketLifeTime"`
}

type Result struct {
	RouterID string `json:"routerID"`
	YourID   string `json:"yourID"`
	Ticket   string `json:"ticket"`
	Offer    *Offer `json:"offer"`
}

func HTTP2Interview[T any](address string, secure bool, resume *Resume[T]) (*Result, error) {
	protocol := "http"
	if secure {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s/wamp/v1/interview", protocol, address)
	result, e := wampShared.JSONPost[Result](url, resume)
	return result, e
}
