package wampInterview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type ErrorPayload struct {
	Message string `json:"message"`
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

func jsonpost(url string, payload any) (response *http.Response, e error) {
	requestBody, e := MakeJSONBuffer(payload)
	if e == nil {
		response, e = http.Post(url, "application/json", requestBody)
		if e == nil {
			return response, nil
		}
	}
	return nil, errors.Join(e, errors.New("failed to send HTTP request"))
}

func JSONPost[T any](url string, inPayload any) (*T, error) {
	response, e := jsonpost(url, inPayload)
	if e == nil {
		outPayload := new(T)
		e = ReadJSONBody(response.Body, outPayload)
		if e == nil {
			return outPayload, nil
		}
	} else if response.StatusCode == 400 {
		responsePayload := new(ErrorPayload)
		e = ReadJSONBody(response.Body, responsePayload)
		if e == nil {
			e = errors.New(responsePayload.Message)
		}
	} else {
		errorMessage := fmt.Sprintf("HTTP2: %s", response.Status)
		e = errors.New(errorMessage)
	}
	return nil, e
}

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
	result, e := JSONPost[SuccessPayload](url, requestPayload)
	return result, e
}
