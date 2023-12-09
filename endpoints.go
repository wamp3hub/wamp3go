package wamp

import (
	"errors"
	"log/slog"
)

var (
	ErrorApplication = errors.New("ApplicationError")
)

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
	var payload T

	v := event.Payload()
	if event.Kind() == MK_ERROR {
		__payload, e := cast[errorEventPayload](v)
		if e == nil {
			e = errors.New(__payload.Message)
		}
		return payload, e
	}

	payload, e := cast[T](v)
	return payload, e
}

type PublishProcedure[I any] func(I, PublishEvent)

type publishEventEndpoint func(PublishEvent)

func NewPublishEventEndpoint[I any](
	procedure PublishProcedure[I],
	__logger *slog.Logger,
) publishEventEndpoint {
	logger := __logger.With("name", "PublishEventEndpoint")
	return func(publishEvent PublishEvent) {
		payload, e := SerializePayload[I](publishEvent)
		if e == nil {
			finally := func() {
				e := recover()
				if e == nil {
					logger.Debug("endpoint execution success")
				} else {
					logger.Debug("during endpoint execution", "error", e)
				}
			}
	
			defer finally()
	
			procedure(payload, publishEvent)
		} else {
			logger.Warn("during serialize payload", "error", e)
		}
	}
}

type generatorExitException struct {
	Source Event
}

func (generatorExitException) Error() string {
	return "GeneratorExit"
}

type CallProcedure[I, O any] func(I, CallEvent) (O, error)

type callEventEndpoint func(CallEvent) ReplyEvent

func NewCallEventEndpoint[I, O any](
	procedure CallProcedure[I, O],
	__logger *slog.Logger,
) callEventEndpoint {
	logger := __logger.With("name", "CallEventEndpoint")
	return func(callEvent CallEvent) (replyEvent ReplyEvent) {
		payload, e := SerializePayload[I](callEvent)
		if e != nil {
			logger.Warn("during serialize payload", "error", e)
			replyEvent = NewErrorEvent(callEvent, InvalidPayload)
			return replyEvent
		}

		var (
			returnValue O
			returnError error
		)

		finally := func() {
			r := recover()
			if r == nil {
				logger.Debug("endpoint execution success")

				switch e := returnError.(type) {
				case nil:
					replyEvent = NewReplyEvent[O](callEvent, returnValue)
				case (*generatorExitException):
					replyEvent = NewErrorEvent(e.Source, e)
				default:
					replyEvent = NewErrorEvent(callEvent, returnError)
				}
			} else {
				logger.Debug("during endpoint execution", "error", r)
				replyEvent = NewErrorEvent(callEvent, ErrorApplication)
			}
		}

		defer finally()

		returnValue, returnError = procedure(payload, callEvent)
		return replyEvent
	}
}
