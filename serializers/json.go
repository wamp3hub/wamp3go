package wampSerializers

import (
	"encoding/json"
	"errors"

	wamp "github.com/wamp3hub/wamp3go"
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

func (JSONSerializer) Encode(event wamp.Event) ([]byte, error) {
	switch event := event.(type) {
	case wamp.AcceptEvent:
		type jsonAcceptMessage struct {
			ID       string               `json:"ID"`
			Kind     wamp.MessageKind     `json:"kind"`
			Features *wamp.AcceptFeatures `json:"features"`
		}
		message := jsonAcceptMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	case wamp.ReplyEvent:
		type jsonReplyMessage struct {
			ID       string              `json:"ID"`
			Kind     wamp.MessageKind    `json:"kind"`
			Features *wamp.ReplyFeatures `json:"features"`
			Payload  any                 `json:"payload"`
		}
		message := jsonReplyMessage{event.ID(), event.Kind(), event.Features(), event.Content()}
		return json.Marshal(message)
	case wamp.PublishEvent:
		type jsonPublishMessage struct {
			ID       string                `json:"ID"`
			Kind     wamp.MessageKind      `json:"kind"`
			Features *wamp.PublishFeatures `json:"features"`
			Payload  any                   `json:"payload"`
			Route    *wamp.PublishRoute    `json:"route"`
		}
		message := jsonPublishMessage{event.ID(), event.Kind(), event.Features(), event.Content(), event.Route()}
		return json.Marshal(message)
	case wamp.CallEvent:
		type jsonCallMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.CallFeatures `json:"features"`
			Payload  any                `json:"payload"`
			Route    *wamp.CallRoute    `json:"route"`
		}
		message := jsonCallMessage{event.ID(), event.Kind(), event.Features(), event.Content(), event.Route()}
		return json.Marshal(message)
	case wamp.NextEvent:
		type jsonNextMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.NextFeatures `json:"features"`
		}
		message := jsonNextMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	}
	return nil, errors.New("InvalidEvent")
}

func (JSONSerializer) Decode(v []byte) (event wamp.Event, e error) {
	type jsonFieldKind struct {
		Kind wamp.MessageKind
	}
	message := new(jsonFieldKind)
	e = json.Unmarshal(v, message)
	if e == nil {
		switch message.Kind {
		case wamp.MK_ACCEPT:
			type jsonAcceptMessage struct {
				ID       string               `json:"ID"`
				Kind     wamp.MessageKind     `json:"kind"`
				Features *wamp.AcceptFeatures `json:"features"`
			}
			message := new(jsonAcceptMessage)
			e = json.Unmarshal(v, &message)
			event = wamp.MakeAcceptEvent(message.ID, message.Features)
		case wamp.MK_REPLY, wamp.MK_ERROR, wamp.MK_YIELD:
			type jsonReplyMessage struct {
				ID       string              `json:"ID"`
				Kind     wamp.MessageKind    `json:"kind"`
				Features *wamp.ReplyFeatures `json:"features"`
				Payload  json.RawMessage     `json:"payload"`
			}
			message := new(jsonReplyMessage)
			e = json.Unmarshal(v, &message)
			event = wamp.MakeReplyEvent(message.ID, message.Kind, message.Features, &jsonPayloadField{message.Payload})
		case wamp.MK_PUBLISH:
			type jsonPublishMessage struct {
				ID       string                `json:"ID"`
				Kind     wamp.MessageKind      `json:"kind"`
				Features *wamp.PublishFeatures `json:"features"`
				Payload  json.RawMessage       `json:"payload"`
				Route    *wamp.PublishRoute    `json:"route"`
			}
			message := jsonPublishMessage{Route: new(wamp.PublishRoute)}
			e = json.Unmarshal(v, &message)
			event = wamp.MakePublishEvent(message.ID, message.Features, &jsonPayloadField{message.Payload}, message.Route)
		case wamp.MK_CALL:
			type jsonCallMessage struct {
				ID       string             `json:"ID"`
				Kind     wamp.MessageKind   `json:"kind"`
				Features *wamp.CallFeatures `json:"features"`
				Payload  json.RawMessage    `json:"payload"`
				Route    *wamp.CallRoute    `json:"route"`
			}
			message := jsonCallMessage{Route: new(wamp.CallRoute)}
			e = json.Unmarshal(v, &message)
			event = wamp.MakeCallEvent(message.ID, message.Features, &jsonPayloadField{message.Payload}, message.Route)
		case wamp.MK_NEXT:
			type jsonNextMessage struct {
				ID       string             `json:"ID"`
				Kind     wamp.MessageKind   `json:"kind"`
				Features *wamp.NextFeatures `json:"features"`
			}
			message := new(jsonNextMessage)
			e = json.Unmarshal(v, &message)
			event = wamp.MakeNextEvent(message.ID, message.Features)
		default:
			e = errors.New("InvalidEvent")
		}
		if e == nil {
			return event, nil
		}
	}
	return nil, e
}

var DefaultSerializer = new(JSONSerializer)
