package wamp

import "errors"

type Encodable[T any] interface {
	Encode() (T, error)
}

type Decodable interface {
	Decode(v any) error
}

type serializableEvent interface {
	ID() string
	Kind() MessageKind
	Payload() any
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

func SerializePayload[T any](event serializableEvent) (T, error) {
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
