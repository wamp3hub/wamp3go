package wamp3go

import (
	"log"
	"time"
)

const DEFAULT_TIMEOUT = time.Minute
const DEFAULT_GENERATOR_LIFETIME = time.Hour

var (
	lastIDMap    map[string]string     = make(map[string]string)
	nextYieldMap map[string]ReplyEvent = make(map[string]ReplyEvent)
)

type Session struct {
	peer          *Peer
	Subscriptions map[string]PublishEndpoint
	Registrations map[string]CallEndpoint
}

func (session *Session) ID() string {
	return session.peer.ID
}

func NewSession(peer *Peer) *Session {
	session := Session{
		peer,
		make(map[string]PublishEndpoint),
		make(map[string]CallEndpoint),
	}

	session.peer.ConsumeIncomingPublishEvents(
		func(publishEvent PublishEvent) {
			route := publishEvent.Route()
			endpoint, found := session.Subscriptions[route.EndpointID]
			if found {
				endpoint(publishEvent)
			} else {
				log.Printf(
					"[session] subscription not found (ID=%s route.ID=%s publisher.ID=%s)",
					session.ID(), route.EndpointID, route.PublisherID,
				)
			}
		},
		func() {},
	)

	session.peer.ConsumeIncomingCallEvents(
		func(callEvent CallEvent) {
			route := callEvent.Route()
			endpoint, found := session.Registrations[route.EndpointID]
			if found {
				replyEvent := endpoint(callEvent)

				generatorID := callEvent.ID()
				nextYieldEvent, found := nextYieldMap[generatorID]
				if found {
					delete(nextYieldMap, generatorID)
					nextEventPromise := peer.PendingNextEvents.New(nextYieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME)
					e := session.peer.Send(nextYieldEvent)
					if e == nil {
						nextEvent, done := <-nextEventPromise
						if done {
							replyEvent = NewReplyEvent(nextEvent, replyEvent.Content())
						}
					}
				}

				e := session.peer.Send(replyEvent)
				if e == nil {
					log.Printf("[session] success call (ID=%s)", session.ID())
				} else {
					log.Printf("[session] reply not sent (ID=%s)", session.ID())
				}
			} else {
				log.Printf(
					"[session] registration not found (ID=%s route.ID=%s caller.ID=%s)",
					session.ID(), route.EndpointID, route.CallerID,
				)
			}
		},
		func() {},
	)

	return &session
}

func (session *Session) Publish(event PublishEvent) error {
	return session.peer.Send(event)
}

func (session *Session) Call(event CallEvent) ReplyEvent {
	replyEventPromise := session.peer.PendingReplyEvents.New(event.ID(), DEFAULT_TIMEOUT)
	e := session.peer.Send(event)
	if e == nil {
		replyEvent, done := <-replyEventPromise
		if done {
			return replyEvent
		}
	}
	return NewErrorEvent(event, e)
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func (session *Session) Subscribe(
	uri string,
	features *SubscribeOptions,
	endpoint PublishEndpoint,
) (*Subscription, error) {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.subscribe"},
		NewResourcePayload[SubscribeOptions]{uri, features},
	)
	replyEvent := session.Call(callEvent)
	if replyEvent.Error() == nil {
		subscription := new(Subscription)
		e := replyEvent.Payload(subscription)
		if e == nil {
			session.Subscriptions[subscription.ID] = endpoint
			return subscription, nil
		}
	}
	return nil, replyEvent.Error()
}

func (session *Session) Register(
	uri string,
	features *RegisterOptions,
	endpoint CallEndpoint,
) (*Registration, error) {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.register"},
		NewResourcePayload[RegisterOptions]{uri, features},
	)
	replyEvent := session.Call(callEvent)
	if replyEvent.Error() == nil {
		registration := new(Registration)
		e := replyEvent.Payload(registration)
		if e == nil {
			session.Registrations[registration.ID] = endpoint
			return registration, nil
		}
	}
	return nil, replyEvent.Error()
}

type DeleteResourcePayload struct {
	ID string
}

func (session *Session) Unsubscribe(subscriptionID string) error {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.unsubscribe"},
		DeleteResourcePayload{subscriptionID},
	)
	replyEvent := session.Call(callEvent)
	if replyEvent.Error() == nil {
		return nil
	}
	return replyEvent.Error()
}

func (session *Session) Unregister(registrationID string) error {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.unregister"},
		DeleteResourcePayload{registrationID},
	)
	replyEvent := session.Call(callEvent)
	if replyEvent.Error() == nil {
		return nil
	}
	return replyEvent.Error()
}

func Yield(callEvent CallEvent, payload any) (e error) {
	generatorID := callEvent.ID()
	nextYieldEvent, exists := nextYieldMap[generatorID]
	if exists {
		peer := callEvent.Peer()
		nextEventPromise := peer.PendingNextEvents.New(nextYieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME)
		e = peer.Send(nextYieldEvent)
		if e == nil {
			nextEvent, done := <-nextEventPromise
			if done {
				nextYieldEvent = newYieldEvent(nextEvent, payload)
				nextYieldMap[generatorID] = nextYieldEvent
			}
		}
	} else {
		nextYieldEvent = newYieldEvent[any](callEvent, nil)
		nextYieldMap[generatorID] = nextYieldEvent
	}
	return e
}

func Next(generator ReplyEvent, timeout time.Duration) ReplyEvent {
	if generator.Kind() != MK_YIELD {
		panic("FirstArgumentMustBeGenerator")
	}

	generatorID := generator.ID()
	lastID, exsits := lastIDMap[generatorID]
	if !exsits {
		lastID = generatorID
	}

	peer := generator.Peer()
	nextEvent := newNextEvent(lastID)
	replyEventPromise := peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	e := peer.Send(nextEvent)
	if e == nil {
		response, done := <-replyEventPromise
		if done {
			if generator.Kind() == MK_YIELD {
				lastIDMap[generatorID] = response.ID()
			} else {
				delete(lastIDMap, generatorID)
			}
			return response
		}
	}

	return NewErrorEvent(nextEvent, e)
}
