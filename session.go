package wamp3go

import (
	"errors"
	"log"
	"time"

	"github.com/wamp3hub/wamp3go/shared"
)

const DEFAULT_TIMEOUT = time.Minute

const DEFAULT_GENERATOR_LIFETIME = time.Hour

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

func Publish[I any](
	session *Session,
	features *PublishFeatures,
	payload I,
) error {
	publishEvent := NewPublishEvent(features, payload)
	e := session.peer.Send(publishEvent)
	return e
}

type PendingResponse[T any] struct {
	promise       shared.Promise[ReplyEvent]
	cancelPromise shared.Cancellable
}

func newPendingResponse[T any](
	promise shared.Promise[ReplyEvent],
	cancelPromise shared.Cancellable,
) *PendingResponse[T] {
	return &PendingResponse[T]{promise, cancelPromise}
}

func (pending *PendingResponse[T]) Cancel() {
	// TODO cancellation
	pending.cancelPromise()
}

func (pending *PendingResponse[T]) Await() (replyEvent ReplyEvent, payload T, e error) {
	// TODO cancellation
	replyEvent, done := <-pending.promise

	if done {
		if replyEvent.Kind() == MK_ERROR {
			__payload := new(errorEventPayload)
			replyEvent.Payload(__payload)
			e = errors.New(__payload.Code)
		} else {
			e = replyEvent.Payload(&payload)
		}
	} else {
		e = errors.New("SomethingWentWrong")
	}
	return replyEvent, payload, e
}

func Call[O, I any](
	session *Session,
	features *CallFeatures,
	payload I,
) *PendingResponse[O] {
	callEvent := NewCallEvent[I](features, payload)
	// TODO cancellation
	replyEventPromise, cancelPromise := session.peer.PendingReplyEvents.New(callEvent.ID(), DEFAULT_TIMEOUT)
	pendingResponse := newPendingResponse[O](replyEventPromise, cancelPromise)
	e := session.peer.Send(callEvent)
	if e != nil {
		pendingResponse.Cancel()
	}
	return pendingResponse
}

type remoteGenerator[T any] struct {
	done                bool
	peer                *Peer
	lastPendingResponse *PendingResponse[T]
}

func (generator *remoteGenerator[T]) Active() bool {
	return !generator.done
}

func NewRemoteGenerator[O, I any](
	session *Session,
	callFeatures *CallFeatures,
	payload I,
) *remoteGenerator[O] {
	pendingResponse := Call[O](session, callFeatures, payload)
	generator := remoteGenerator[O]{false, session.peer, pendingResponse}
	return &generator
}

func (generator *remoteGenerator[T]) Next(timeout time.Duration) (response ReplyEvent, outPayload T, e error) {
	if generator.done {
		panic("GeneratorExit")
	}
	response, outPayload, e = generator.lastPendingResponse.Await()
	if response.Kind() != MK_YIELD {
		generator.done = true
		return response, outPayload, e
	}
	nextEvent := newNextEvent(response)
	// TODO cancellation
	replyEventPromise, cancelPromise := generator.peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	generator.lastPendingResponse = newPendingResponse[T](replyEventPromise, cancelPromise)
	e = generator.peer.Send(nextEvent)
	return response, outPayload, e
}

func (generator *remoteGenerator[T]) Stop() error {
	generator.done = true
	// TODO cancellation
	return nil
}

func Yield[I any](
	source Event,
	inPayload I,
) (nextEvent NextEvent, e error) {
	yieldEvent := newYieldEvent(source, inPayload)
	peer := source.getPeer()
	nextEventPromise, _ := peer.PendingNextEvents.New(yieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME)
	e = peer.Send(yieldEvent)
	if e == nil {
		nextEvent, done := <-nextEventPromise
		if done {
			return nextEvent, nil
		}
		e = errors.New("TimedOut")
	}
	return nil, e
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func Subscribe(
	session *Session,
	uri string,
	options *SubscribeOptions,
	endpoint PublishEndpoint,
) (*Subscription, error) {
	pendingResponse := Call[Subscription](
		session,
		&CallFeatures{"wamp.subscribe"},
		NewResourcePayload[SubscribeOptions]{uri, options},
	)
	_, subscription, e := pendingResponse.Await()
	if e == nil {
		session.Subscriptions[subscription.ID] = endpoint
		return &subscription, nil
	}
	return nil, e
}

func Register(
	session *Session,
	uri string,
	options *RegisterOptions,
	endpoint CallEndpoint,
) (*Registration, error) {
	pendingResponse := Call[Registration](
		session,
		&CallFeatures{"wamp.register"},
		NewResourcePayload[RegisterOptions]{uri, options},
	)
	_, registration, e := pendingResponse.Await()
	if e == nil {
		session.Registrations[registration.ID] = endpoint
		return &registration, nil
	}
	return nil, e
}

func Unsubscribe(
	session *Session,
	subscriptionID string,
) error {
	pendingResponse := Call[struct{}](session, &CallFeatures{"wamp.unsubscribe"}, subscriptionID)
	_, _, e := pendingResponse.Await()
	if e == nil {
		delete(session.Subscriptions, subscriptionID)
	}
	return e
}

func Unregister(
	session *Session,
	registrationID string,
) error {
	pendingResponse := Call[struct{}](session, &CallFeatures{"wamp.unregister"}, registrationID)
	_, _, e := pendingResponse.Await()
	if e == nil {
		delete(session.Registrations, registrationID)
	}
	return e
}

func Leave(
	session *Session,
	reason string,
) error {
	// TODO
	return nil
}
