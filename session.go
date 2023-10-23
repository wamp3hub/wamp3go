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

				nextYieldEvent := callEvent.getNextYield()
				if nextYieldEvent != nil {
					nextEventPromise, _ := peer.PendingNextEvents.New(nextYieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME)
					e := session.peer.Say(nextYieldEvent)
					if e == nil {
						nextEvent, done := <-nextEventPromise
						if done {
							replyEvent = NewReplyEvent(nextEvent, replyEvent.Content())
						}
					}
				}

				e := session.peer.Say(replyEvent)
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
	e := session.peer.Say(publishEvent)
	return e
}

type PendingResult[T any] struct {
	promise       shared.Promise[ReplyEvent]
	cancelPromise shared.Cancellable
	callbackList  []func(ReplyEvent)
}

func NewPendingResult[T any](
	promise shared.Promise[ReplyEvent],
	cancelPromise shared.Cancellable,
) *PendingResult[T] {
	return &PendingResult[T]{promise, cancelPromise, []func(ReplyEvent){}}
}

func (pending *PendingResult[T]) addCallback(f func(ReplyEvent)) {
	pending.callbackList = append(pending.callbackList, f)
}

func (pending *PendingResult[T]) Cancel() {
	pending.cancelPromise()
}

func (pending *PendingResult[T]) Await() (replyEvent ReplyEvent, payload T, e error) {
	replyEvent, done := <-pending.promise

	for _, callback := range pending.callbackList {
		callback(replyEvent)
	}

	if done {
		if replyEvent.Kind() == MK_ERROR {
			replyEvent.Payload(e)
		} else {
			e = replyEvent.Payload(&payload)
		}
	} else {
		e = errors.New("InternalException")
	}
	return replyEvent, payload, e
}

func Call[O, I any](
	session *Session,
	features *CallFeatures,
	payload I,
) *PendingResult[O] {
	callEvent := NewCallEvent[I](features, payload)
	// TODO cancellation
	replyEventPromise, cancelPromise := session.peer.PendingReplyEvents.New(callEvent.ID(), DEFAULT_TIMEOUT)
	pendingResult := NewPendingResult[O](replyEventPromise, cancelPromise)
	e := session.peer.Say(callEvent)
	if e != nil {
		pendingResult.Cancel()
	}
	return pendingResult
}

type generator[T any] struct {
	done   bool
	nextID string
	peer   *Peer
}

func (g *generator[T]) Done() bool {
	return g.done
}

func (g *generator[T]) onYield(response ReplyEvent) {
	if response.Kind() == MK_YIELD {
		g.nextID = response.ID()
	} else {
		g.done = true
	}
}

func (g *generator[T]) Next(timeout time.Duration) *PendingResult[T] {
	nextEvent := newNextEvent(g.nextID)
	// TODO cancellation
	replyEventPromise, cancelPromise := g.peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	pendingResult := NewPendingResult[T](replyEventPromise, cancelPromise)
	pendingResult.addCallback(g.onYield)
	e := g.peer.Say(nextEvent)
	if e != nil {
		pendingResult.Cancel()
	}
	return pendingResult
}

func (g *generator[T]) Stop() error {
	g.done = true
	// TODO cancellation
	return nil
}

func NewGenerator[O, I any](
	session *Session,
	callFeatures *CallFeatures,
	payload I,
) (*generator[O], error) {
	result := Call[struct{}](session, callFeatures, payload)
	yieldEvent, _, e := result.Await()

	if yieldEvent.Kind() != MK_YIELD {
		e = errors.New("ProcedureIsNotGenerator")
	}

	if e == nil {
		g := generator[O]{false, yieldEvent.ID(), session.peer}
		return &g, nil
	}

	return nil, e
}

func Yield[I any](callEvent CallEvent, payload I) error {
	nextYieldEvent := callEvent.getNextYield()
	if nextYieldEvent == nil {
		nextYieldEvent = newYieldEvent(callEvent, struct{}{})
		callEvent.setNextYield(nextYieldEvent)
		return nil
	}

	peer := callEvent.getPeer()
	nextEventPromise, _ := peer.PendingNextEvents.New(nextYieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME)
	e := peer.Say(nextYieldEvent)
	if e == nil {
		nextEvent, done := <-nextEventPromise
		if done {
			nextYieldEvent = newYieldEvent(nextEvent, payload)
			callEvent.setNextYield(nextYieldEvent)
		}
	}

	return e
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func Subscribe(
	session *Session,
	uri string,
	features *SubscribeOptions,
	endpoint PublishEndpoint,
) (*Subscription, error) {
	result := Call[Subscription](
		session,
		&CallFeatures{"wamp.subscribe"},
		NewResourcePayload[SubscribeOptions]{uri, features},
	)
	_, subscription, e := result.Await()
	if e == nil {
		session.Subscriptions[subscription.ID] = endpoint
		return &subscription, nil
	}
	return nil, e
}

func Register(
	session *Session,
	uri string,
	features *RegisterOptions,
	endpoint CallEndpoint,
) (*Registration, error) {
	result := Call[Registration](
		session,
		&CallFeatures{"wamp.register"},
		NewResourcePayload[RegisterOptions]{uri, features},
	)
	_, registration, e := result.Await()
	if e == nil {
		session.Registrations[registration.ID] = endpoint
		return &registration, nil
	}
	return nil, e
}

type DeleteResourcePayload struct {
	ID string
}

func Unsubscribe(
	session *Session,
	subscriptionID string,
) error {
	result := Call[any](session, &CallFeatures{"wamp.unsubscribe"}, DeleteResourcePayload{subscriptionID})
	_, _, e := result.Await()
	return e
}

func Unregister(
	session *Session,
	registrationID string,
) error {
	result := Call[any](session, &CallFeatures{"wamp.unregister"}, DeleteResourcePayload{registrationID})
	_, _, e := result.Await()
	return e
}

func Leave(
	session *Session,
	reason string,
) error {
	// TODO
	return nil
}
