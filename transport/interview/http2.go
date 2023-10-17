package interview

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type Payload struct {
	Credentials any `json:"credentials"`
}

type ErrorPayload struct {
	Code string `json:"code"`
}

type SuccessPayload struct {
	RouterID string `json:"routerID"`
	YourID   string `json:"yourID"`
	Ticket   string `json:"ticket"`
}

func HTTP2Interview(address string, requestPayload *Payload) (*SuccessPayload, error) {
	requestBodyBytes, e := json.Marshal(requestPayload)
	if e == nil {
		requestBody := bytes.NewBuffer(requestBodyBytes)
		url := "http://" + address + "/wamp3/interview"
		request, _ := http.NewRequest("POST", url, requestBody)
		request.Header.Set("Content-Type", "application/json")
		client := new(http.Client)
		response, e := client.Do(request)
		if e == nil {
			responseBody, e := io.ReadAll(response.Body)
			if e == nil {
				response.Body.Close()
				if response.StatusCode == 200 {
					responsePayload := SuccessPayload{}
					e = json.Unmarshal(responseBody, &responsePayload)
					if e == nil {
						return &responsePayload, nil
					}
				} else {
					responsePayload := ErrorPayload{}
					e = json.Unmarshal(responseBody, &responsePayload)
					if e == nil {
						e = errors.New(responsePayload.Code)
					}
				}
			}
		}
	}
	return nil, e
}
