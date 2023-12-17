package wampInterview

import (
	"fmt"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

type Payload struct {
	Credentials any `json:"credentials"`
}

type SuccessPayload struct {
	RouterID string `json:"routerID"`
	YourID   string `json:"yourID"`
	Ticket   string `json:"ticket"`
}

func HTTP2Interview(address string, secure bool, requestPayload *Payload) (*SuccessPayload, error) {
	protocol := "http"
	if secure {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s/wamp/v1/interview", protocol, address)
	result, e := wampShared.JSONPost[SuccessPayload](url, requestPayload)
	return result, e
}
