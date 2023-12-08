package wamp

import (
	"errors"
	"log/slog"
)

var (
	ErrorApplication = errors.New("ApplicationError")
)

type Decodable interface {
	Decode(v any) error
}

type eventSerializable interface {
	Event
	eventPayload
}

func serializePayload[T any](event eventSerializable) (*T, error) {
	v := event.Payload()
	decoder, ok := v.(Decodable)

	if event.Kind() == MK_ERROR {
		if ok {
			payload := new(errorEventPayload)
			e := decoder.Decode(payload)
			if e == nil {
				return nil, errors.New(payload.Message)
			}
		}

		payload, ok := v.(errorEventPayload)
		if ok {
			return nil, errors.New(payload.Message)
		}

		return nil, errors.New("unexpected type")
	}

	if ok {
		payload := new(T)
		e := decoder.Decode(payload)
		return payload, e
	}

	payload, ok := v.(T)
	if ok {
		return &payload, nil
	}

	return nil, errors.New("unexpected type")
}

type PublishProcedure[I any] func(I, PublishEvent)

type publishEventEndpoint func(PublishEvent)

func NewPublishEventEndpoint[I any](
	procedure PublishProcedure[I],
	__logger *slog.Logger,
) publishEventEndpoint {
	logger := __logger.With("name", "PublishEventEndpoint")
	return func(publishEvent PublishEvent) {
		payload, e := serializePayload[I](publishEvent)
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
	
			procedure(*payload, publishEvent)
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
		payload, e := serializePayload[I](callEvent)
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

		returnValue, returnError = procedure(*payload, callEvent)
		return replyEvent
	}
}
