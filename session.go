package wamp3go

import (
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
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

var __lastYieldMap = make(map[string]string)

func (session *Session) Yield(callEvent CallEvent, payload any) (e error) {
	generatorID := callEvent.ID()
	lastYield, exists := __lastYieldMap[generatorID]
	if !exists {
		yieldEvent := NewYieldEvent[any](generatorID, nil)
		lastYield = yieldEvent.ID()
		acceptEventPromise := session.peer.PendingAcceptEvents.New(lastYield, DEFAULT_TIMEOUT)
		session.peer.Transport.Send(yieldEvent)
		<-acceptEventPromise
	}

	nextEventPromise := session.peer.PendingNextEvents.New(lastYield, DEFAULT_GENERATOR_LIFETIME)
	nextEvent, done := <-nextEventPromise
	if done {
		acceptEvent := NewAcceptEvent(nextEvent.ID())
		session.peer.Transport.Send(acceptEvent)

		yieldEvent := NewYieldEvent(nextEvent.ID(), payload)
		lastYield = yieldEvent.ID()
		acceptEventPromise := session.peer.PendingAcceptEvents.New(lastYield, DEFAULT_TIMEOUT)
		session.peer.Transport.Send(yieldEvent)
		_, done := <-acceptEventPromise
		if done {
			__lastYieldMap[generatorID] = lastYield
		}
	}

	return e
}

var lastYieldMap = make(map[string]string)

func (session *Session) Next(yieldEvent ReplyEvent, timeout time.Duration) ReplyEvent {
	if yieldEvent.Kind() != MK_YIELD {
		panic("FirstArgumentMustBeGenerator")
	}

	generatorID := yieldEvent.ID()
	lastYield, exsits := lastYieldMap[generatorID]
	if !exsits {
		lastYield = generatorID
	}

	nextEvent := NewNextEvent(lastYield)
	acceptEventPromise := session.peer.PendingAcceptEvents.New(nextEvent.ID(), DEFAULT_TIMEOUT)
	replyEventPromise := session.peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	session.peer.Transport.Send(nextEvent)
	_, done := <-acceptEventPromise
	if done {
		response, done := <-replyEventPromise
		if done {
			if yieldEvent.Kind() == MK_YIELD {
				lastYieldMap[generatorID] = response.ID()
			} else {
				delete(lastYieldMap, generatorID)
			}
			return response
		}
	}

	return NewErrorEvent(uuid.NewString(), errors.New("SomethingWentWrong"))
}

type Session struct {
	peer          *Peer
	Subscriptions map[string]publishEndpoint
	Registrations map[string]callEndpoint
}

func (session *Session) ID() string {
	return session.peer.ID
}

func NewSession(peer *Peer) *Session {
	session := Session{peer, make(map[string]publishEndpoint), make(map[string]callEndpoint)}

	session.peer.IncomingPublishEvents.Consume(
		func(publishEvent PublishEvent) {
			acceptEvent := NewAcceptEvent(publishEvent.ID())
			session.peer.Transport.Send(acceptEvent)

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
			acceptEvent := NewAcceptEvent(callEvent.ID())
			session.peer.Transport.Send(acceptEvent)

			route := callEvent.Route()
			endpoint, exist := session.Registrations[route.EndpointID]
			if exist {
				replyEvent := endpoint(callEvent)

				delete(__lastYieldMap, callEvent.ID())

				acceptEventPromise := session.peer.PendingAcceptEvents.New(replyEvent.ID(), DEFAULT_TIMEOUT)
				session.peer.Transport.Send(replyEvent)
				_, done := <-acceptEventPromise
				if done {

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
	acceptEventPromise := session.peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	session.peer.Transport.Send(event)
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return errors.New("SomethingWentWrong")
}

func (session *Session) Call(event CallEvent) (replyEvent ReplyEvent) {
	acceptEventPromise := session.peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	replyEventPromise := session.peer.PendingReplyEvents.New(event.ID(), DEFAULT_TIMEOUT)
	session.peer.Transport.Send(event)
	_, done := <-acceptEventPromise
	if done {
		replyEvent, done = <-replyEventPromise
		if done {
			acceptEvent := NewAcceptEvent(replyEvent.ID())
			session.peer.Transport.Send(acceptEvent)
			return replyEvent
		}
	}
	return NewErrorEvent(event.ID(), errors.New("SomethingWentWrong"))
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
