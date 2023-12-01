package wamp

type PublishProcedure func(PublishEvent)

type CallProcedure func(CallEvent) ReplyEvent

type publishEventEndpoint func(PublishEvent)

func NewPublishEventEndpoint(procedure PublishProcedure) publishEventEndpoint {
	return func(publishEvent PublishEvent) {
		finnaly := func() {
			e := recover()
			if e == nil {
			} else {
			}
		}

		defer finnaly()

		procedure(publishEvent)
	}
}

type callEventEndpoint func(CallEvent) ReplyEvent

func NewCallEventEndpoint(procedure CallProcedure) callEventEndpoint {
	return func(callEvent CallEvent) (replyEvent ReplyEvent) {
		finnaly := func() {
			e := recover()
			if e == nil {

			} else {
				replyEvent = NewReplyEvent(callEvent, e)
			}
		}

		defer finnaly()

		replyEvent = procedure(callEvent)
		return replyEvent
	}
}
