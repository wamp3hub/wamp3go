package wamp


type PublishProcedure func(PublishEvent)

type CallProcedure func(CallEvent) ReplyEvent

type publishEndpoint struct {
	procedure PublishProcedure
}

func NewPublishEndpoint(procedure PublishProcedure) *publishEndpoint {
	return &publishEndpoint{procedure}
}

func (endpoint publishEndpoint) execute(event PublishEvent) {
	defer recover()
	endpoint.procedure(event)
}

type callEndpoint struct {
	procedure CallProcedure
	result ReplyEvent
}

func NewCallEndpoint(procedure CallProcedure) *callEndpoint {
	return &callEndpoint{procedure, nil}
}

func (endpoint callEndpoint) recover(callEvent CallEvent) {
	r := recover()
	if r == nil {
		// TODO Log
	} else {
		// TODO Log
		endpoint.result = NewReplyEvent(callEvent, InternalError)
	}
}

func (endpoint callEndpoint) execute(callEvent CallEvent) ReplyEvent {
	defer endpoint.recover(callEvent)
	endpoint.result = endpoint.procedure(callEvent)
	return endpoint.result
}
