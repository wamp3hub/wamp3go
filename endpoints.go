package wamp

import (
	"errors"
	"log/slog"
)

var (
	ErrorApplication = errors.New("application error")
	ErrorInvalidPayload = errors.New("invalid payload")
)

type ProcedureToPublish[I any] func(I, PublishEvent)

type publishEventEndpoint func(PublishEvent)

func NewPublishEventEndpoint[I any](
	procedure ProcedureToPublish[I],
	__logger *slog.Logger,
) publishEventEndpoint {
	logger := __logger.With("name", "PublishEventEndpoint")
	return func(publishEvent PublishEvent) {
		payload, e := ReadPayload[I](publishEvent)
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

type ProcedureToCall[I, O any] func(I, CallEvent) (O, error)

type callEventEndpoint func(CallEvent) ReplyEvent

func NewCallEventEndpoint[I, O any](
	procedure ProcedureToCall[I, O],
	__logger *slog.Logger,
) callEventEndpoint {
	logger := __logger.With("name", "CallEventEndpoint")
	return func(callEvent CallEvent) (replyEvent ReplyEvent) {
		payload, e := ReadPayload[I](callEvent)
		if e != nil {
			logger.Warn("during serialize payload", "error", e)
			replyEvent = NewErrorEvent(callEvent, ErrorInvalidPayload)
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
				case *generatorExitException:
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
