package wamp

import (
	"errors"
	"log/slog"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

const DEFAULT_GENERATOR_LIFETIME = 3600

var ErrorDispatch = errors.New("dispatch error")

type Session struct {
	Subscriptions map[string]publishEventEndpoint
	Registrations map[string]callEventEndpoint
	restorables   map[string]func()
	router        *Peer
	logger        *slog.Logger
}

func (session *Session) ID() string {
	return session.router.ID
}

func NewSession(
	router *Peer,
	logger *slog.Logger,
) *Session {
	session := Session{
		make(map[string]publishEventEndpoint),
		make(map[string]callEventEndpoint),
		make(map[string]func()),
		router,
		logger.With(
			slog.Group("session", "ID", router.ID),
		),
	}

	session.router.IncomingPublishEvents.Observe(
		func(publishEvent PublishEvent) {
			features := publishEvent.Features()
			route := publishEvent.Route()

			logData := slog.Group(
				"publishEvent",
				"ID", publishEvent.ID(),
				"URI", features.URI,
				"Route.EndpointID", route.EndpointID,
				"Route.PublisherID", route.PublisherID,
			)
			session.logger.Debug("incoming publish event", logData)

			endpoint, exists := session.Subscriptions[route.EndpointID]
			if exists {
				endpoint(publishEvent)
			} else {
				session.logger.Error("subscription not found", logData)
				// TODO handle
			}
		},
		func() {
			clear(session.Subscriptions)
		},
	)

	session.router.IncomingCallEvents.Observe(
		func(callEvent CallEvent) {
			features := callEvent.Features()
			route := callEvent.Route()

			callEventLogData := slog.Group(
				"callEvent",
				"ID", callEvent.ID(),
				"URI", features.URI,
				"Route.EndpointID", route.EndpointID,
				"Route.CallerID", route.CallerID,
			)
			session.logger.Debug("incoming call event", callEventLogData)

			endpoint, exists := session.Registrations[route.EndpointID]
			if exists {
				replyEvent := endpoint(callEvent)

				replyEventLogData := slog.Group("replyEvent", "ID", replyEvent.ID())

				ok := session.router.Send(replyEvent, DEFAULT_RESEND_COUNT)
				if ok {
					session.logger.Debug("call event processed successfully", callEventLogData, replyEventLogData)
				} else {
					session.logger.Error("reply event dispatch error", callEventLogData, replyEventLogData)
				}
			} else {
				session.logger.Error("registration not found", callEventLogData)
				// TODO handle
			}
		},
		func() {
			clear(session.Registrations)
		},
	)

	session.router.ReJoinEvents.Observe(
		func(__ struct{}) {
			for name, restore := range session.restorables {
				session.logger.Debug("restoring", "name", name)
				restore()
			}
		},
		func() {
			clear(session.restorables)
		},
	)

	return &session
}

func Publish[I any](
	session *Session,
	features *PublishFeatures,
	payload I,
) error {
	publishEvent := newPublishEvent(features, payload)

	logData := slog.Group(
		"publishEvent",
		"ID", publishEvent.ID(),
		"URI", features.URI,
	)

	session.logger.Debug("trying to send call event", logData)
	ok := session.router.Send(publishEvent, DEFAULT_RESEND_COUNT)
	if ok {
		session.logger.Debug("publication successfully sent", logData)
		return nil
	}

	session.logger.Error("publish event dispatch error", logData)
	return ErrorDispatch
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

func (pendingResponse *PendingResponse[T]) lock() {
	if pendingResponse.used {
		panic("can not use again")
	}
	pendingResponse.used = true
}

func (pendingResponse *PendingResponse[T]) Cancel() {
	pendingResponse.lock()
	pendingResponse.cancelPromise()
}

func (pendingResponse *PendingResponse[T]) Await() (ReplyEvent, T, error) {
	pendingResponse.lock()

	replyEvent, promiseCompleted := <-pendingResponse.promise
	if promiseCompleted {
		payload, e := ReadPayload[T](replyEvent)
		return replyEvent, payload, e
	}

	var __ T
	return nil, __, ErrorTimedOut
}

func Call[O, I any](
	session *Session,
	features *CallFeatures,
	payload I,
) *PendingResponse[O] {
	if features.Timeout == 0 {
		features.Timeout = DEFAULT_TIMEOUT
	}

	callEvent := newCallEvent[I](features, payload)

	logData := slog.Group(
		"callEvent",
		"ID", callEvent.ID(),
		"URI", features.URI,
		"Timeout", features.Timeout,
	)

	replyTimeout := time.Duration(2*features.Timeout) * time.Second
	replyEventPromise, cancelReplyEventPromise := session.router.PendingReplyEvents.New(
		callEvent.ID(), replyTimeout,
	)

	cancelCallEvent := func() {
		session.logger.Debug("trying to cancel invocation", logData)
		cancelEvent := newCancelEvent(callEvent)
		ok := session.router.Send(cancelEvent, DEFAULT_RESEND_COUNT)
		if ok {
			session.logger.Debug("invocation successfully cancelled", logData)
		} else {
			session.logger.Error("call event dispatch error", logData)
		}
		cancelReplyEventPromise()
	}

	pendingResponse := newPendingResponse[O](replyEventPromise, cancelCallEvent)

	session.logger.Debug("trying to send call event", logData)
	ok := session.router.Send(callEvent, DEFAULT_RESEND_COUNT)
	if ok {
		session.logger.Debug("call event successfully sent", logData)
	} else {
		session.logger.Error("call event dispatch error", logData)
		errorEvent := NewErrorEvent(callEvent, ErrorDispatch)
		session.router.PendingReplyEvents.Complete(callEvent.ID(), errorEvent)
	}

	return pendingResponse
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func Subscribe[I any](
	session *Session,
	uri string,
	options *SubscribeOptions,
	procedure ProcedureToPublish[I],
) (*Subscription, error) {
	logData := slog.Group("subscription", "URI", uri)
	session.logger.Debug("trying to subscribe", logData)

	pendingResponse := Call[*Subscription](
		session,
		&CallFeatures{URI: "wamp.router.subscribe"},
		NewResourcePayload[SubscribeOptions]{uri, options},
	)

	_, subscription, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewPublishEventEndpoint[I](procedure, session.logger)
		session.Subscriptions[subscription.ID] = endpoint
		session.logger.Debug("new subscription", logData)

		session.restorables[subscription.ID] = func() {
			delete(session.Subscriptions, subscription.ID)
			Subscribe[I](session, uri, options, procedure)
		}

		return subscription, nil
	}

	session.logger.Error("during subscribe", "error", e, logData)
	return nil, e
}

func Register[I, O any](
	session *Session,
	uri string,
	options *RegisterOptions,
	procedure ProcedureToCall[I, O],
) (*Registration, error) {
	logData := slog.Group("registration", "URI", uri)
	session.logger.Debug("trying to register", logData)

	pendingResponse := Call[*Registration](
		session,
		&CallFeatures{URI: "wamp.router.register"},
		NewResourcePayload[RegisterOptions]{uri, options},
	)

	_, registration, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewCallEventEndpoint[I, O](procedure, session.logger)
		session.Registrations[registration.ID] = endpoint
		session.logger.Debug("new registration", logData)

		session.restorables[registration.ID] = func() {
			delete(session.Registrations, registration.ID)
			Register[I, O](session, uri, options, procedure)
		}

		return registration, nil
	}

	session.logger.Error("during register", "error", e, logData)
	return nil, e
}

func Unsubscribe(
	session *Session,
	subscriptionID string,
) error {
	logData := slog.Group("subscription", "ID", subscriptionID)
	session.logger.Debug("trying to unsubscribe", logData)
	pendingResponse := Call[struct{}](
		session,
		&CallFeatures{URI: "wamp.router.unsubscribe"},
		subscriptionID,
	)
	_, _, e := pendingResponse.Await()
	if e == nil {
		delete(session.Subscriptions, subscriptionID)
		delete(session.restorables, subscriptionID)
		session.logger.Debug("subscription successfully dettached", logData)
	}
	session.logger.Error("during unsubscribe", "error", e, logData)
	return e
}

func Unregister(
	session *Session,
	registrationID string,
) error {
	logData := slog.Group("registration", "ID", registrationID)
	session.logger.Debug("trying to unregister", logData)
	pendingResponse := Call[struct{}](
		session,
		&CallFeatures{URI: "wamp.router.unregister"},
		registrationID,
	)
	_, _, e := pendingResponse.Await()
	if e == nil {
		delete(session.Registrations, registrationID)
		delete(session.restorables, registrationID)
		session.logger.Debug("registration successfully dettached", logData)
	}
	session.logger.Error("during unregister", "error", e, logData)
	return e
}

func Leave(
	session *Session,
	reason string,
) error {
	logData := slog.Group("leave", "reason", reason)
	session.logger.Debug("trying to leave", logData)
	e := session.router.Close()
	if e == nil {
		session.logger.Debug("session successfully left", logData)
	} else {
		session.logger.Error("during leave", "error", e, logData)
	}
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
	logger      *slog.Logger
}

func (generator *remoteGenerator[T]) Active() bool {
	return !generator.done
}

func NewRemoteGenerator[O, I any](
	session *Session,
	features *CallFeatures,
	inPayload I,
) (*remoteGenerator[O], error) {
	logData := slog.Group("generator", "URI", features.URI)
	session.logger.Debug("trying to initialize remote generator", logData)
	pendingResponse := Call[NewGeneratorPayload](session, features, inPayload)
	yieldEvent, generator, e := pendingResponse.Await()
	if e == nil {
		logData = slog.Group("generator", "ID", generator.ID, "URI", features.URI)
		instance := remoteGenerator[O]{
			false,
			generator.ID,
			yieldEvent.ID(),
			session.router,
			session.logger.With("name", "RemoteGenerator", logData),
		}
		session.logger.Debug("remote generator successfully initialized", logData)
		return &instance, nil
	}

	session.logger.Error("during initialize remote generator", "error", e, logData)
	return nil, e
}

func (generator *remoteGenerator[T]) Next(
	timeout uint64,
) (response ReplyEvent, outPayload T, e error) {
	logData := slog.Group(
		"nextEvent",
		"yieldID", generator.lastYieldID,
		"timeout", timeout,
	)
	generator.logger.Debug("trying to get next", logData)

	if generator.done {
		generator.logger.Error("generator already done", logData)
		panic("generator exit")
	}

	nextFeatures := NextFeatures{generator.ID, generator.lastYieldID, timeout}
	nextEvent := newNextEvent(&nextFeatures)
	responseTimeout := time.Duration(2*timeout) * time.Second
	responsePromise, cancelResponsePromise := generator.peer.PendingReplyEvents.New(nextEvent.ID(), responseTimeout)
	pendingResponse := newPendingResponse[T](responsePromise, cancelResponsePromise)
	ok := generator.peer.Send(nextEvent, DEFAULT_RESEND_COUNT)
	if !ok {
		generator.logger.Error("next event dispatch error", logData)
		cancelResponsePromise()
		generator.done = true
		return nil, outPayload, ErrorDispatch
	}

	response, outPayload, e = pendingResponse.Await()
	if e == nil && response.Kind() == MK_YIELD {
		generator.logger.Debug("yield event successfully received", logData)
		generator.lastYieldID = response.ID()
	} else {
		generator.logger.Debug("destroying generator", "error", e, logData)
		generator.done = true
		// TODO handle
	}

	return response, outPayload, e
}

func (generator *remoteGenerator[T]) Stop() error {
	generator.logger.Debug("trying to stop generator")

	if generator.done {
		generator.logger.Error("generator already done")
		panic("generator exit")
	}

	stopEvent := NewStopEvent(generator.ID)
	ok := generator.peer.Send(stopEvent, DEFAULT_RESEND_COUNT)
	if ok {
		generator.done = true
		generator.logger.Debug("generator successfully stopped")
		return nil
	}

	generator.logger.Error("generator stop event dispatch error")
	return ErrorDispatch
}

func yieldNext(
	router *Peer,
	generatorID string,
	lifetime time.Duration,
	yieldEvent YieldEvent,
	__logger *slog.Logger,
) NextEvent {
	logger := __logger.With(
		slog.Group(
			"yieldEvent",
			"ID", yieldEvent.ID(),
			"GeneratorID", generatorID,
			"GeneratorLifetime", lifetime,
		),
	)

	nextEventPromise, cancelNextEventPromise := router.PendingNextEvents.New(yieldEvent.ID(), 0)

	stopEventPromise, cancelStopEventPromise := router.PendingCancelEvents.New(generatorID, lifetime)

	logger.Debug("trying to send yield event")
	ok := router.Send(yieldEvent, DEFAULT_RESEND_COUNT)
	if !ok {
		logger.Error("yield event dispatch error (destroying generator)")
		cancelNextEventPromise()
		cancelStopEventPromise()
		panic("protocol error")
	}

	select {
	case _, done := <-stopEventPromise:
		if done {
			logger.Warn("generator stop event received (destroying generator)")
		} else {
			logger.Warn("generator lifetime expired (destroying generator)")
		}
		cancelNextEventPromise()
		panic("generator destroy")
	case nextEvent := <-nextEventPromise:
		cancelStopEventPromise()
		logger.Debug("generator next", "nextEvent.ID", nextEvent.ID())
		return nextEvent
	}
}

func Yield[I any](
	source Event,
	inPayload I,
) NextEvent {
	router := source.getPeer()
	logger := router.logger.With(
		"name", "Yield",
		"sourceEvent.Kind", source.Kind(),
	)

	lifetime := DEFAULT_GENERATOR_LIFETIME * time.Second

	callEvent, ok := source.(CallEvent)
	if ok {
		generator := NewGeneratorPayload{wampShared.NewID()}
		yieldEvent := newYieldEvent(callEvent, generator)
		source = yieldNext(router, generator.ID, lifetime, yieldEvent, logger)
	}

	nextEvent, ok := source.(NextEvent)
	if ok {
		nextFeatures := nextEvent.Features()
		yieldEvent := newYieldEvent(nextEvent, inPayload)
		return yieldNext(router, nextFeatures.GeneratorID, lifetime, yieldEvent, logger)
	}

	logger.Error("invalid source event (destroying generator)")
	panic("invalid source event")
}

type generatorExitException struct {
	Source Event
}

func (generatorExitException) Error() string {
	return "GeneratorExit"
}

func GeneratorExit(source Event) *generatorExitException {
	return &generatorExitException{source}
}
