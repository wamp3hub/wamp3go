package wamp

import "errors"

type Decodable interface {
	Decode(v any) error
}

func cast[T any](v any) (T, error) {
	var payload T

	decoder, ok := v.(Decodable)
	if ok {
		e := decoder.Decode(&payload)
		return payload, e
	}

	__payload, ok := v.(T)
	if ok {
		return __payload, nil
	}

	return payload, errors.New("unexpected type")
}

type readableEventPayload interface {
	ID() string
	Kind() MessageKind
	Payload() any
}

func ReadPayload[T any](event readableEventPayload) (T, error) {
	v := event.Payload()
	if event.Kind() == MK_ERROR {
		__payload, e := cast[errorEventPayload](v)
		if e == nil {
			e = errors.New(__payload.Message)
		}
		var __ T
		return __, e
	}

	payload, e := cast[T](v)
	return payload, e
}
