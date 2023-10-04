package wamp3go

import (
	"errors"
	"log"
	"time"
)

const DEFAULT_TIMEOUT = time.Minute
const DEFAULT_GENERATOR_LIFETIME = time.Hour

func ExtractError(event ReplyEvent) error {
	payload := new(ErrorEventPayload)
	e := event.Payload(payload)
	if e == nil {
		return errors.New(payload.Code)
	}
	return e
}

type Session struct {
	peer          *Peer
	Subscriptions map[string]publishEndpoint
	Registrations map[string]callEndpoint
	lastIDMap     map[string]string
	lastYieldMap  map[string]ReplyEvent
}

func (session *Session) ID() string {
	return session.peer.ID
}

func NewSession(peer *Peer) *Session {
	session := Session{
		peer,
		make(map[string]publishEndpoint),
		make(map[string]callEndpoint),
		make(map[string]string),
		make(map[string]ReplyEvent),
	}

	session.peer.IncomingPublishEvents.Consume(
		func(publishEvent PublishEvent) {
			route := publishEvent.Route()
			endpoint, exist := session.Subscriptions[route.EndpointID]
			if exist {
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

	session.peer.IncomingCallEvents.Consume(
		func(callEvent CallEvent) {
			route := callEvent.Route()
			endpoint, exist := session.Registrations[route.EndpointID]
			if exist {
				replyEvent := endpoint(callEvent)

				lastYield, found := session.lastYieldMap[callEvent.ID()]
				if found {
					e := session.peer.Send(lastYield)
					if e == nil {
						delete(session.lastYieldMap, callEvent.ID())
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
	return NewErrorEvent(event.ID(), e)
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func (session *Session) Subscribe(
	uri string,
	features *SubscribeOptions,
	endpoint publishEndpoint,
) (*Subscription, error) {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.subscribe"},
		&NewResourcePayload[SubscribeOptions]{uri, features},
	)
	replyEvent := session.Call(callEvent)
	replyFeatures := replyEvent.Features()
	if replyFeatures.OK {
		subscription := new(Subscription)
		e := replyEvent.Payload(subscription)
		if e == nil {
			session.Subscriptions[subscription.ID] = endpoint
			return subscription, nil
		}
	}
	return nil, ExtractError(replyEvent)
}

func (session *Session) Register(
	uri string,
	features *RegisterOptions,
	endpoint callEndpoint,
) (*Registration, error) {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.register"},
		&NewResourcePayload[RegisterOptions]{uri, features},
	)
	replyEvent := session.Call(callEvent)
	replyFeatures := replyEvent.Features()
	if replyFeatures.OK {
		registration := new(Registration)
		e := replyEvent.Payload(registration)
		if e == nil {
			session.Registrations[registration.ID] = endpoint
			return registration, nil
		}
	}
	return nil, ExtractError(replyEvent)
}

type DeleteResourcePayload struct {
	ID string
}

func (session *Session) Unsubscribe(subscriptionID string) error {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.unsubscribe"},
		&DeleteResourcePayload{subscriptionID},
	)
	replyEvent := session.Call(callEvent)
	replyFeatures := replyEvent.Features()
	if replyFeatures.OK {
		return nil
	}
	return ExtractError(replyEvent)
}

func (session *Session) Unregister(registrationID string) error {
	callEvent := NewCallEvent(
		&CallFeatures{"wamp.unregister"},
		&DeleteResourcePayload{registrationID},
	)
	replyEvent := session.Call(callEvent)
	replyFeatures := replyEvent.Features()
	if replyFeatures.OK {
		return nil
	}
	return ExtractError(replyEvent)
}

func (session *Session) Yield(callEvent CallEvent, payload any) (e error) {
	generatorID := callEvent.ID()
	lastYield, exists := session.lastYieldMap[generatorID]
	if exists {
		nextEventPromise := session.peer.PendingNextEvents.New(lastYield.ID(), DEFAULT_GENERATOR_LIFETIME)
		e = session.peer.Send(lastYield)
		if e == nil {
			nextEvent, done := <-nextEventPromise
			if done {
				lastYield = NewYieldEvent(nextEvent.ID(), payload)
				session.lastYieldMap[generatorID] = lastYield
			}
		}
	} else {
		lastYield = NewYieldEvent[any](generatorID, nil)
		session.lastYieldMap[generatorID] = lastYield
	}
	return e
}

func (session *Session) Next(yieldEvent ReplyEvent, timeout time.Duration) ReplyEvent {
	if yieldEvent.Kind() != MK_YIELD {
		panic("FirstArgumentMustBeGenerator")
	}

	generatorID := yieldEvent.ID()
	lastID, exsits := session.lastIDMap[generatorID]
	if !exsits {
		lastID = generatorID
	}

	nextEvent := NewNextEvent(lastID)
	replyEventPromise := session.peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	e := session.peer.Send(nextEvent)
	if e == nil {
		response, done := <-replyEventPromise
		if done {
			if yieldEvent.Kind() == MK_YIELD {
				session.lastIDMap[generatorID] = response.ID()
			} else {
				delete(session.lastYieldMap, generatorID)
			}
			return response
		}
	}

	return NewErrorEvent(nextEvent.ID(), e)
}
