package wamp

import (
	"errors"
	"log/slog"
)

var (
	ErrorApplication = errors.New("ApplicationError")
)

type PublishProcedure func(PublishEvent)

type CallProcedure func(CallEvent) any

type publishEventEndpoint func(PublishEvent)

func NewPublishEventEndpoint(
	procedure PublishProcedure,
	logger *slog.Logger,
) publishEventEndpoint {
	return func(publishEvent PublishEvent) {
		finally := func() {
			e := recover()
			if e == nil {
				logger.Debug("endpoint execution success")
			} else {
				logger.Debug("during endpoint execution", "error", e)
			}
		}

		defer finally()

		procedure(publishEvent)
	}
}

type callEventEndpoint func(CallEvent) ReplyEvent

func NewCallEventEndpoint[O any](
	procedure CallProcedure,
	logger *slog.Logger,
) callEventEndpoint {
	return func(callEvent CallEvent) (replyEvent ReplyEvent) {
		var returning any

		finally := func() {
			e := recover()
			if e == nil {
				logger.Debug("endpoint execution success")

				e, isError := returning.(error)
				if isError {
					replyEvent = NewErrorEvent(callEvent, e)
				} else {
					__returning := returning.(O)
					replyEvent = NewReplyEvent(callEvent, __returning)
				}
			} else {
				logger.Debug("during endpoint execution", "error", e)
				replyEvent = NewErrorEvent(callEvent, ErrorApplication)
			}
		}

		defer finally()

		returning = procedure(callEvent)
		return replyEvent
	}
}
