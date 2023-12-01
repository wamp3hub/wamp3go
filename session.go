package wamp

import (
	"errors"
	"log"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var (
	TimedOutError      = errors.New("TimedOut")
	CancelledError     = errors.New("Cancelled")
	SomethingWentWrong = errors.New("SomethingWentWrong")
)

const DEFAULT_TIMEOUT = time.Minute

const DEFAULT_GENERATOR_LIFETIME = time.Hour

type Session struct {
	peer          *Peer
	Subscriptions map[string]publishEventEndpoint
	Registrations map[string]callEventEndpoint
}

func (session *Session) ID() string {
	return session.peer.ID
}

func NewSession(peer *Peer) *Session {
	session := Session{
		peer,
		make(map[string]publishEventEndpoint),
		make(map[string]callEventEndpoint),
	}

	session.peer.IncomingPublishEvents.Observe(
		func(publishEvent PublishEvent) {
			route := publishEvent.Route()
			endpoint, exists := session.Subscriptions[route.EndpointID]
			if exists {
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

	session.peer.IncomingCallEvents.Observe(
		func(callEvent CallEvent) {
			route := callEvent.Route()
			endpoint, exists := session.Registrations[route.EndpointID]
			if exists {
				// TODO cancellation

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
	publishEvent := newPublishEvent(features, payload)
	e := session.peer.Send(publishEvent)
	return e
}

type PendingResponse[T any] struct {
	used          bool
	promise       wampShared.Promise[ReplyEvent]
	cancelPromise wampShared.CancelPromise
}

func newPendingResponse[T any](
	promise wampShared.Promise[ReplyEvent],
	cancelPromise wampShared.CancelPromise,
) *PendingResponse[T] {
	return &PendingResponse[T]{false, promise, cancelPromise}
}

func (pendingResponse *PendingResponse[T]) markAsUsed() {
	if pendingResponse.used {
		panic("can not use again")
	}
	pendingResponse.used = true
}

func (pendingResponse *PendingResponse[T]) Cancel() {
	pendingResponse.markAsUsed()
	pendingResponse.cancelPromise()
}

func (pendingResponse *PendingResponse[T]) Await() (replyEvent ReplyEvent, payload T, e error) {
	pendingResponse.markAsUsed()

	replyEvent, promiseCompleted := <-pendingResponse.promise

	if promiseCompleted {
		if replyEvent.Kind() == MK_ERROR {
			__payload := new(errorEventPayload)
			replyEvent.Payload(__payload)
			e = errors.New(__payload.Message)
		} else {
			e = replyEvent.Payload(&payload)
		}
	} else {
		e = TimedOutError
	}
	return replyEvent, payload, e
}

func Call[O, I any](
	session *Session,
	features *CallFeatures,
	payload I,
) (*PendingResponse[O], error) {
	if features.Timeout == 0 {
		features.Timeout = DEFAULT_TIMEOUT
	}

	callEvent := newCallEvent[I](features, payload)
	replyEventPromise, cancelPromise := session.peer.PendingReplyEvents.New(
		callEvent.ID(),
		time.Duration(features.Timeout),
	)

	cancelCallEvent := func() {
		cancelEvent := newCancelEvent(callEvent)
		e := session.peer.Send(cancelEvent)
		if e == nil {
			cancelPromise()
		} else {
			log.Printf(
				"[session] failed to send cancel event (ID=%s event.ID=%s)", session.ID(), callEvent.ID(),
			)
		}
	}

	pendingResponse := newPendingResponse[O](replyEventPromise, cancelCallEvent)
	e := session.peer.Send(callEvent)
	if e == nil {
		return pendingResponse, nil
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
	procedure PublishProcedure,
) (*Subscription, error) {
	pendingResponse, e := Call[Subscription](
		session,
		&CallFeatures{URI: "wamp.router.subscribe"},
		NewResourcePayload[SubscribeOptions]{uri, options},
	)
	if e != nil {
		// TODO log
		return nil, e
	}

	_, subscription, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewPublishEventEndpoint(procedure)
		session.Subscriptions[subscription.ID] = endpoint
		return &subscription, nil
	}
	return nil, e
}

func Register(
	session *Session,
	uri string,
	options *RegisterOptions,
	procedure CallProcedure,
) (*Registration, error) {
	pendingResponse, e := Call[Registration](
		session,
		&CallFeatures{URI: "wamp.router.register"},
		NewResourcePayload[RegisterOptions]{uri, options},
	)
	if e != nil {
		// TODO log
		return nil, e
	}

	_, registration, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewCallEventEndpoint(procedure)
		session.Registrations[registration.ID] = endpoint
		return &registration, nil
	}
	return nil, e
}

func Unsubscribe(
	session *Session,
	subscriptionID string,
) error {
	pendingResponse, e := Call[struct{}](
		session,
		&CallFeatures{URI: "wamp.router.unsubscribe"},
		subscriptionID,
	)
	if e == nil {
		delete(session.Subscriptions, subscriptionID)
		_, _, e = pendingResponse.Await()
	}
	return e
}

func Unregister(
	session *Session,
	registrationID string,
) error {
	pendingResponse, e := Call[struct{}](
		session,
		&CallFeatures{URI: "wamp.router.unregister"},
		registrationID,
	)
	if e == nil {
		delete(session.Registrations, registrationID)
		_, _, e = pendingResponse.Await()
	}
	return e
}

func Leave(
	session *Session,
	reason string,
) error {
	e := session.peer.Close()
	return e
}

type NewGeneratorPayload struct {
	ID string `json:"id"`
}

type remoteGenerator[T any] struct {
	done        bool
	ID          string
	lastYieldID string
	peer        *Peer
}

func (generator *remoteGenerator[T]) Active() bool {
	return !generator.done
}

func NewRemoteGenerator[O, I any](
	session *Session,
	callFeatures *CallFeatures,
	inPayload I,
) (*remoteGenerator[O], error) {
	pendingResponse, e := Call[NewGeneratorPayload](session, callFeatures, inPayload)
	if e != nil {
		// TODO log
		return nil, e
	}

	yieldEvent, generator, e := pendingResponse.Await()
	if e == nil {
		instance := remoteGenerator[O]{false, generator.ID, yieldEvent.ID(), session.peer}
		return &instance, nil
	}
	return nil, e
}

func (generator *remoteGenerator[T]) Next(
	timeout time.Duration,
) (response ReplyEvent, outPayload T, e error) {
	if generator.done {
		panic("generator exit")
	}

	nextFeatures := NextFeatures{generator.ID, generator.lastYieldID, timeout}
	nextEvent := newNextEvent(&nextFeatures)
	replyEventPromise, cancelPromise := generator.peer.PendingReplyEvents.New(nextEvent.ID(), timeout)
	pendingResponse := newPendingResponse[T](replyEventPromise, cancelPromise)
	e = generator.peer.Send(nextEvent)
	if e == nil {
		response, outPayload, e = pendingResponse.Await()
		if e == nil && response.Kind() == MK_YIELD {
			generator.lastYieldID = response.ID()
		} else {
			generator.done = true
		}
	}
	return response, outPayload, e
}

func (generator *remoteGenerator[T]) Stop() error {
	if generator.done {
		panic("generator exit")
	}

	generator.done = true
	stopEvent := NewStopEvent(generator.ID)
	e := generator.peer.Send(stopEvent)
	return e
}

func Yield[I any](
	source Event,
	inPayload I,
) (NextEvent, error) {
	peer := source.getPeer()

	callEvent, ok := source.(CallEvent)
	if ok {
		generator := NewGeneratorPayload{wampShared.NewID()}
		yieldEvent := newYieldEvent(callEvent, generator)
		nextEventPromise, _ := peer.PendingNextEvents.New(
			yieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME,
		)
		e := peer.Send(yieldEvent)
		if e != nil {
			// TODO log
			return nil, e
		}

		nextEvent, done := <-nextEventPromise
		if done {
			source = nextEvent
		} else {
			return nil, TimedOutError
		}
	}

	nextEvent, ok := source.(NextEvent)
	if !ok {
		panic("invalid source event")
	}

	nextFeatures := nextEvent.Features()

	stopEventPromise, cancelStopEventPromise := peer.PendingCancelEvents.New(
		nextFeatures.GeneratorID, DEFAULT_GENERATOR_LIFETIME,
	)

	yieldEvent := newYieldEvent(nextEvent, inPayload)

	nextEventPromise, cancelNextEventPromise := peer.PendingNextEvents.New(
		yieldEvent.ID(), DEFAULT_GENERATOR_LIFETIME,
	)

	e := peer.Send(yieldEvent)
	if e != nil {
		// TODO log
		return nil, e
	}

	select {
	case <-stopEventPromise:
		cancelNextEventPromise()

		return nil, CancelledError
	case nextEvent := <-nextEventPromise:
		cancelStopEventPromise()

		return nextEvent, nil
	}
}
