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

var yieldMap = make(map[string]string)

func (session *Session) Yield(callEvent CallEvent, payload any) (e error) {
	generatorID := callEvent.ID()
	lastYield, exists := yieldMap[generatorID]
	if !exists {
		yieldEvent := NewYieldEvent[any](generatorID, nil)
		e = session.peer.Transport.Send(yieldEvent)
		if e == nil {
			// TODO acknowledgment
			lastYield = yieldEvent.ID()
			yieldMap[generatorID] = lastYield
		}
	}

	nextEvent, e := session.peer.PendingNextEvents.Catch(lastYield, DEFAULT_GENERATOR_LIFETIME)
	// TODO acknowledgment
	if e == nil {
		yieldEvent := NewYieldEvent(nextEvent.ID(), payload)
		e = session.peer.Transport.Send(yieldEvent)
		if e == nil {
			// TODO acknowledgment
			lastYield = yieldEvent.ID()
			yieldMap[generatorID] = lastYield
		}
	}

	return e
}

var nextMap = make(map[string]string)

func (session *Session) Next(yieldEvent ReplyEvent, timeout time.Duration) ReplyEvent {
	if yieldEvent.Kind() != MK_YIELD {
		panic("FirstArgumentMustBeGenerator")
	}

	generatorID := yieldEvent.ID()
	lastNext, exsits := nextMap[generatorID]
	if !exsits {
		lastNext = generatorID
	}
	nextEvent := NewNextEvent(lastNext)
	e := session.peer.Transport.Send(nextEvent)
	if e == nil {
		// TODO acknowledgment
		response, e := session.peer.PendingReplyEvents.Catch(nextEvent.ID(), timeout)
		if e == nil {
			if yieldEvent.Kind() == MK_YIELD {
				nextMap[generatorID] = response.ID()
			} else {
				delete(nextMap, generatorID)
			}
			return response
		}
	}
	return NewErrorEvent(uuid.NewString(), e)
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
			e := session.peer.Transport.Send(acceptEvent)
			if e != nil {
				log.Printf("[session] accept not sent (ID=%s e=%s)", session.ID(), e)
			}
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
			// acceptEvent := NewAcceptEvent(callEvent.ID())
			// e := session.peer.Transport.Send(acceptEvent)
			// if e != nil {
			// 	log.Printf("[session] accept not sent (ID=%s e=%s)", session.ID(), e)
			// }
			route := callEvent.Route()
			endpoint, exist := session.Registrations[route.EndpointID]
			if exist {
				replyEvent := endpoint(callEvent)
				e := session.peer.Transport.Send(replyEvent)
				if e == nil {
					delete(yieldMap, replyEvent.ID())
				} else {
					log.Printf("[session] reply not sent (ID=%s e=%s)", session.ID(), e)
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
	e := session.peer.Transport.Send(event)
	if e == nil {
		_, e = session.peer.PendingAcceptEvents.Catch(event.ID(), DEFAULT_TIMEOUT)
	}
	return e
}

func (session *Session) Call(event CallEvent) (replyEvent ReplyEvent) {
	e := session.peer.Transport.Send(event)
	if e == nil {
		replyEvent, e = session.peer.PendingReplyEvents.Catch(event.ID(), DEFAULT_TIMEOUT)
		if e == nil {
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
