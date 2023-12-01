package wampInterview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func MakeJSONBuffer(body any) (*bytes.Buffer, error) {
	bodyBytes, e := json.Marshal(body)
	if e == nil {
		buffer := bytes.NewBuffer(bodyBytes)
		return buffer, nil
	}
	return nil, e
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
	protocol := "http"
	if secure {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s/wamp/v1/interview", protocol, address)

	requestBody, e := MakeJSONBuffer(requestPayload)
	if e != nil {
		return nil, errors.Join(e, errors.New("failed to marshal request payload"))
	}

	response, e := http.Post(url, "application/json", requestBody)
	if e != nil {
		return nil, errors.Join(e, errors.New("failed to send HTTP request"))
	}

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
	} else {
		e = errors.New("interview " + response.Status)
	}

	return nil, e
}