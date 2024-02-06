package wampShared

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

func jsonPostRequest(url string, payload any) (response *http.Response, e error) {
	requestBody, e := MakeJSONBuffer(payload)
	if e == nil {
		response, e = http.Post(url, "application/json", requestBody)
		if e == nil {
			return response, nil
		}
	}
	return nil, errors.Join(e, errors.New("during send HTTP request"))
}

func JSONPost[T any](url string, inPayload any) (*T, error) {
	response, e := jsonPostRequest(url, inPayload)
	if e == nil && response.StatusCode == 200 {
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
