package wampSerializers

import (
	"encoding/json"
	"errors"
	"fmt"

	wamp "github.com/wamp3hub/wamp3go"
)

type JSONPayloadField struct {
	Value json.RawMessage
}

func (field *JSONPayloadField) Decode(v any) error {
	return json.Unmarshal(field.Value, v)
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
		message := jsonReplyMessage{event.ID(), event.Kind(), event.Features(), event.Payload()}
		return json.Marshal(message)
	case wamp.PublishEvent:
		type jsonPublishMessage struct {
			ID       string                `json:"ID"`
			Kind     wamp.MessageKind      `json:"kind"`
			Features *wamp.PublishFeatures `json:"features"`
			Payload  any                   `json:"payload"`
			Route    *wamp.PublishRoute    `json:"route"`
		}
		message := jsonPublishMessage{event.ID(), event.Kind(), event.Features(), event.Payload(), event.Route()}
		return json.Marshal(message)
	case wamp.CallEvent:
		type jsonCallMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.CallFeatures `json:"features"`
			Payload  any                `json:"payload"`
			Route    *wamp.CallRoute    `json:"route"`
		}
		message := jsonCallMessage{event.ID(), event.Kind(), event.Features(), event.Payload(), event.Route()}
		return json.Marshal(message)
	case wamp.NextEvent:
		type jsonNextMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.NextFeatures `json:"features"`
		}
		message := jsonNextMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	case wamp.CancelEvent:
		type jsonCancelMessage struct {
			ID       string              `json:"ID"`
			Kind     wamp.MessageKind    `json:"kind"`
			Features *wamp.ReplyFeatures `json:"features"`
		}
		message := jsonCancelMessage{event.ID(), event.Kind(), event.Features()}
		return json.Marshal(message)
	}

	return nil, errors.New("UnexpectedEventKind")
}

func (JSONSerializer) getMessageKind(v []byte) (wamp.MessageKind, error) {
	type jsonFieldKind struct {
		Kind wamp.MessageKind `json:"kind"`
	}
	message := new(jsonFieldKind)
	e := json.Unmarshal(v, message)
	if e == nil {
		return message.Kind, nil
	}
	return wamp.MK_UNDEFINED, e
}

func (serializer JSONSerializer) Decode(v []byte) (event wamp.Event, e error) {
	messageKind, e := serializer.getMessageKind(v)

	switch messageKind {
	case wamp.MK_ACCEPT:
		type jsonAcceptMessage struct {
			ID       string               `json:"ID"`
			Kind     wamp.MessageKind     `json:"kind"`
			Features *wamp.AcceptFeatures `json:"features"`
		}
		message := new(jsonAcceptMessage)
		e = json.Unmarshal(v, message)
		event = wamp.MakeAcceptEvent(message.ID, message.Features)
		return event, e
	case wamp.MK_REPLY, wamp.MK_ERROR, wamp.MK_YIELD:
		type jsonReplyMessage struct {
			ID       string              `json:"ID"`
			Kind     wamp.MessageKind    `json:"kind"`
			Features *wamp.ReplyFeatures `json:"features"`
			Payload  json.RawMessage     `json:"payload"`
		}
		message := new(jsonReplyMessage)
		e = json.Unmarshal(v, message)
		event = wamp.MakeReplyEvent(message.ID, message.Kind, message.Features, &JSONPayloadField{message.Payload})
		return event, e
	case wamp.MK_PUBLISH:
		type jsonPublishMessage struct {
			ID       string                `json:"ID"`
			Kind     wamp.MessageKind      `json:"kind"`
			Features *wamp.PublishFeatures `json:"features"`
			Payload  json.RawMessage       `json:"payload"`
			Route    *wamp.PublishRoute    `json:"route"`
		}
		message := new(jsonPublishMessage)
		e = json.Unmarshal(v, message)
		if message.Route == nil {
			message.Route = new(wamp.PublishRoute)
		}
		event = wamp.MakePublishEvent(message.ID, message.Features, &JSONPayloadField{message.Payload}, message.Route)
		return event, e
	case wamp.MK_CALL:
		type jsonCallMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.CallFeatures `json:"features"`
			Payload  json.RawMessage    `json:"payload"`
			Route    *wamp.CallRoute    `json:"route"`
		}
		message := new(jsonCallMessage)
		e = json.Unmarshal(v, message)
		if message.Route == nil {
			message.Route = new(wamp.CallRoute)
		}
		event = wamp.MakeCallEvent(message.ID, message.Features, &JSONPayloadField{message.Payload}, message.Route)
		return event, e
	case wamp.MK_NEXT:
		type jsonNextMessage struct {
			ID       string             `json:"ID"`
			Kind     wamp.MessageKind   `json:"kind"`
			Features *wamp.NextFeatures `json:"features"`
		}
		message := new(jsonNextMessage)
		e = json.Unmarshal(v, message)
		event = wamp.MakeNextEvent(message.ID, message.Features)
		return event, e
	case wamp.MK_CANCEL:
		type jsonCancelMessage struct {
			ID       string              `json:"ID"`
			Kind     wamp.MessageKind    `json:"kind"`
			Features *wamp.ReplyFeatures `json:"features"`
		}
		message := new(jsonCancelMessage)
		e = json.Unmarshal(v, message)
		event = wamp.MakeCancelEvent(message.ID, message.Features)
		return event, e
	}

	errorMessage := fmt.Sprintf("UnexpectedEventKind(%d)", messageKind)
	e = errors.New(errorMessage)
	return nil, e
}

var DefaultSerializer = new(JSONSerializer)
