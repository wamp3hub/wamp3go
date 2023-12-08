package wamp

import (
	"errors"
	"log/slog"
)

var (
	ErrorApplication = errors.New("ApplicationError")
)

type PublishProcedure func(PublishEvent)

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

type CallProcedure func(CallEvent) any

type generatorExitException struct {
	Source Event
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

				switch __returning := returning.(type) {
				case O:
					replyEvent = NewReplyEvent(callEvent, __returning)
				case error:
					replyEvent = NewErrorEvent(callEvent, __returning)
				case *generatorExitException:
					replyEvent = NewErrorEvent(__returning.Source, errors.New("GeneratorExit"))
				default:
					logger.Debug("during endpoint execution", "error", e)
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
