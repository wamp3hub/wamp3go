package wampInterview

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

func ReadJSONBody(body io.ReadCloser, v any) error {
	bodyBytes, e := io.ReadAll(body)
	if e == nil {
		body.Close()
		e = json.Unmarshal(bodyBytes, v)
	}
	return e
}

func HTTP2Interview(address string, secure bool, requestPayload *Payload) (*SuccessPayload, error) {
	requestBodyBytes, e := json.Marshal(requestPayload)
	if e == nil {
		requestBody := bytes.NewBuffer(requestBodyBytes)
		url := "http://" + address + "/wamp/v1/interview"
		response, _ := http.Post(url, "application/json", requestBody)
		if response.StatusCode == 200 {
			responsePayload := new(SuccessPayload)
			e = ReadJSONBody(response.Body, responsePayload)
			if e == nil {
				return responsePayload, nil
			}
		} else if response.StatusCode == 400 {
			responsePayload := new(ErrorPayload)
			e = ReadJSONBody(response.Body, responsePayload)
			if e == nil {
				return nil, errors.New(responsePayload.Code)
			}
		}

		e = errors.New("interview " + response.Status)
	}
	return nil, e
}
