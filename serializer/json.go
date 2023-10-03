package serializer

import (
	"encoding/json"
	"errors"

	client "github.com/wamp3hub/wamp3go"
)

type jsonPayloadField struct {
	payload json.RawMessage
}

func (field *jsonPayloadField) Content() any {
	return field.payload
}

func (field *jsonPayloadField) Payload(v any) error {
	return json.Unmarshal(field.payload, v)
}

type JSONSerializer struct{}

func (JSONSerializer) Code() string {
	return "json"
}

func (JSONSerializer) Encode(event client.Event) ([]byte, error) {
	switch event := event.(type) {
	case client.AcceptEvent:
		type jsonAcceptMessage struct {
			ID       string                 `json:"ID"`
			Kind     client.MessageKind     `json:"kind"`
			Features *client.AcceptFeatures `json:"features"`
		}
		message := jsonAcceptMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	case client.ReplyEvent:
		type jsonReplyMessage struct {
			ID       string                `json:"ID"`
			Kind     client.MessageKind    `json:"kind"`
			Features *client.ReplyFeatures `json:"features"`
			Payload  any                   `json:"payload"`
		}
		message := jsonReplyMessage{event.ID(), event.Kind(), event.Features(), event.Content()}
		return json.Marshal(message)
	case client.PublishEvent:
		type jsonPublishMessage struct {
			ID       string                  `json:"ID"`
			Kind     client.MessageKind      `json:"kind"`
			Features *client.PublishFeatures `json:"features"`
			Payload  any                     `json:"payload"`
			Route    *client.PublishRoute    `json:"route"`
		}
		message := jsonPublishMessage{event.ID(), event.Kind(), event.Features(), event.Content(), event.Route()}
		return json.Marshal(message)
	case client.CallEvent:
		type jsonCallMessage struct {
			ID       string               `json:"ID"`
			Kind     client.MessageKind   `json:"kind"`
			Features *client.CallFeatures `json:"features"`
			Payload  any                  `json:"payload"`
			Route    *client.CallRoute    `json:"route"`
		}
		message := jsonCallMessage{event.ID(), event.Kind(), event.Features(), event.Content(), event.Route()}
		return json.Marshal(message)
	case client.NextEvent:
		type jsonNextMessage struct {
			ID       string               `json:"ID"`
			Kind     client.MessageKind   `json:"kind"`
			Features *client.NextFeatures `json:"features"`
		}
		message := jsonNextMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	}
	return nil, errors.New("InvalidEvent")
}

func (JSONSerializer) Decode(v []byte) (event client.Event, e error) {
	type jsonFieldKind struct {
		Kind client.MessageKind
	}
	message := new(jsonFieldKind)
	e = json.Unmarshal(v, message)
	if e == nil {
		switch message.Kind {
		case client.MK_ACCEPT:
			type jsonAcceptMessage struct {
				ID       string                 `json:"ID"`
				Kind     client.MessageKind     `json:"kind"`
				Features *client.AcceptFeatures `json:"features"`
			}
			message := new(jsonAcceptMessage)
			e = json.Unmarshal(v, &message)
			event = client.MakeAcceptEvent(message.ID, message.Features)
		case client.MK_REPLY, client.MK_YIELD:
			type jsonReplyMessage struct {
				ID       string                `json:"ID"`
				Kind     client.MessageKind    `json:"kind"`
				Features *client.ReplyFeatures `json:"features"`
				Payload  json.RawMessage       `json:"payload"`
			}
			message := new(jsonReplyMessage)
			e = json.Unmarshal(v, &message)
			event = client.MakeReplyEvent(message.ID, message.Kind, message.Features, &jsonPayloadField{message.Payload})
		case client.MK_PUBLISH:
			type jsonPublishMessage struct {
				ID       string                  `json:"ID"`
				Kind     client.MessageKind      `json:"kind"`
				Features *client.PublishFeatures `json:"features"`
				Payload  json.RawMessage         `json:"payload"`
				Route    *client.PublishRoute    `json:"route"`
			}
			message := jsonPublishMessage{Route: new(client.PublishRoute)}
			e = json.Unmarshal(v, &message)
			event = client.MakePublishEvent(message.ID, message.Features, &jsonPayloadField{message.Payload}, message.Route)
		case client.MK_CALL:
			type jsonCallMessage struct {
				ID       string               `json:"ID"`
				Kind     client.MessageKind   `json:"kind"`
				Features *client.CallFeatures `json:"features"`
				Payload  json.RawMessage      `json:"payload"`
				Route    *client.CallRoute    `json:"route"`
			}
			message := jsonCallMessage{Route: new(client.CallRoute)}
			e = json.Unmarshal(v, &message)
			event = client.MakeCallEvent(message.ID, message.Features, &jsonPayloadField{message.Payload}, message.Route)
		case client.MK_NEXT:
			type jsonNextMessage struct {
				ID       string               `json:"ID"`
				Kind     client.MessageKind   `json:"kind"`
				Features *client.NextFeatures `json:"features"`
			}
			message := new(jsonNextMessage)
			e = json.Unmarshal(v, &message)
			event = client.MakeNextEvent(message.ID, message.Features)
		default:
			e = errors.New("InvalidEvent")
		}
		if e == nil {
			return event, nil
		}
	}
	return nil, e
}

var DefaultJSONSerializer JSONSerializer
