package join

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

type JoinPayload struct {
	Serializer string `json:"serializer"`
}

type JoinSuccessPayload struct {
	PeerID string `json:"peerID"`
	Token  string `json:"token"`
}

type JoinErrorPayload struct {
	Code string `json:"code"`
}

func HTTP2Join(address string, requestPayload *JoinPayload) (*JoinSuccessPayload, error) {
	requestBodyBytes, e := json.Marshal(requestPayload)
	if e == nil {
		requestBody := bytes.NewBuffer(requestBodyBytes)
		url := "http://" + address + "/wamp3/gateway"
		log.Printf("join request (url=%s)", url)
		request, _ := http.NewRequest("POST", url, requestBody)
		request.Header.Set("Content-Type", "application/json")
		client := new(http.Client)
		response, e := client.Do(request)
		if e == nil {
			responseBody, e := io.ReadAll(response.Body)
			if e == nil {
				response.Body.Close()
				if response.StatusCode == 200 {
					responsePayload := JoinSuccessPayload{}
					e = json.Unmarshal(responseBody, &responsePayload)
					if e == nil {
						return &responsePayload, nil
					}
				} else {
					responsePayload := JoinErrorPayload{}
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
