package wamp

import (
	"errors"
	"log/slog"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

const (
	DEFAULT_TIMEOUT            = 60
	DEFAULT_GENERATOR_LIFETIME = 3600
)

var (
	ErrorDispatch  = errors.New("dispatch error")
	ErrorTimedOut  = errors.New("timedout")
	ErrorCancelled = errors.New("cancelled")
)

type Session struct {
	Subscriptions map[string]publishEventEndpoint
	Registrations map[string]callEventEndpoint
	restores      map[string]func()
	peer          *Peer
	logger        *slog.Logger
}

func (session *Session) ID() string {
	return session.peer.Details.ID
}

func (session *Session) Role() string {
	return session.peer.Details.Role
}

func NewSession(
	peer *Peer,
	logger *slog.Logger,
) *Session {
	session := Session{
		make(map[string]publishEventEndpoint),
		make(map[string]callEventEndpoint),
		make(map[string]func()),
		peer,
		logger.With(
			slog.Group(
				"session",
				"ID", peer.Details.ID,
			),
		),
	}

	session.peer.IncomingPublishEvents.Observe(
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

	session.peer.IncomingCallEvents.Observe(
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

				ok := session.peer.Send(replyEvent, DEFAULT_RESEND_COUNT)
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

	session.peer.RejoinEvents.Observe(
		func(__ struct{}) {
			for name, restore := range session.restores {
				session.logger.Debug("restoring", "name", name)
				restore()
			}
		},
		func() {
			clear(session.restores)
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

	__logger := session.logger.With(
		slog.Group(
			"publishEvent",
			"ID", publishEvent.ID(),
			"URI", features.URI,
		),
	)

	if len(features.IncludeRoles) == 0 {
		__logger.Warn("fill in the list of allowed subscribers or roles for security reasons")
	}

	__logger.Debug("trying to send call event")
	ok := session.peer.Send(publishEvent, DEFAULT_RESEND_COUNT)
	if ok {
		__logger.Debug("publication successfully sent")
		return nil
	}

	__logger.Error("publish event dispatch error")
	return ErrorDispatch
}

type PendingResponse[T any] struct {
	done          bool
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
	if pendingResponse.done {
		panic("can not use again")
	}
	pendingResponse.done = true
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

	__logger := session.logger.With(
		slog.Group(
			"callEvent",
			"ID", callEvent.ID(),
			"URI", features.URI,
			"Timeout", features.Timeout,
		),
	)

	if len(features.IncludeRoles) == 0 {
		__logger.Warn("fill in the list of allowed executor roles for security reasons")
	}

	replyTimeout := time.Duration(features.Timeout)*time.Second + time.Second
	replyEventPromise, cancelReplyEventPromise := session.peer.PendingReplyEvents.New(
		callEvent.ID(), replyTimeout,
	)

	cancelCallEvent := func() {
		__logger.Debug("trying to cancel invocation")
		cancelEvent := NewCancelEvent(callEvent)
		ok := session.peer.Send(cancelEvent, DEFAULT_RESEND_COUNT)
		if ok {
			__logger.Debug("invocation successfully cancelled")
		} else {
			__logger.Error("call event dispatch error")
		}
		cancelReplyEventPromise()
	}

	pendingResponse := newPendingResponse[O](replyEventPromise, cancelCallEvent)

	__logger.Debug("trying to send call event")
	ok := session.peer.Send(callEvent, DEFAULT_RESEND_COUNT)
	if ok {
		__logger.Debug("call event successfully sent")
	} else {
		__logger.Error("call event dispatch error")
		errorEvent := NewErrorEvent(callEvent, ErrorDispatch)
		session.peer.PendingReplyEvents.Complete(callEvent.ID(), errorEvent)
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
	__logger := session.logger.With(
		slog.Group(
			"subscription",
			"URI", uri,
		),
	)
	__logger.Debug("trying to subscribe")

	if len(options.IncludeRoles) == 0 {
		__logger.Warn("fill in the list of allowed publisher roles for security reasons")
	}

	pendingResponse := Call[*Subscription](
		session,
		&CallFeatures{URI: "wamp.router.subscribe", IncludeRoles: []string{"router"}},
		NewResourcePayload[SubscribeOptions]{uri, options},
	)

	_, subscription, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewPublishEventEndpoint[I](procedure, __logger)
		session.Subscriptions[subscription.ID] = endpoint
		__logger.Debug("new subscription")

		onRejoin := func() {
			delete(session.Subscriptions, subscription.ID)
			Subscribe[I](session, uri, options, procedure)
		}
		session.restores[subscription.ID] = onRejoin

		return subscription, nil
	}

	__logger.Error("during subscribe", "error", e)
	return nil, e
}

func Register[I, O any](
	session *Session,
	uri string,
	options *RegisterOptions,
	procedure ProcedureToCall[I, O],
) (*Registration, error) {
	__logger := session.logger.With(
		slog.Group(
			"registration",
			"URI", uri,
		),
	)
	__logger.Debug("trying to register")

	if len(options.IncludeRoles) == 0 {
		__logger.Warn("fill in the list of allowed caller roles for security reasons")
	}

	pendingResponse := Call[*Registration](
		session,
		&CallFeatures{URI: "wamp.router.register", IncludeRoles: []string{"router"}},
		NewResourcePayload[RegisterOptions]{uri, options},
	)

	_, registration, e := pendingResponse.Await()
	if e == nil {
		endpoint := NewCallEventEndpoint[I, O](procedure, __logger)
		session.Registrations[registration.ID] = endpoint
		__logger.Debug("new registration")

		onRejoin := func() {
			delete(session.Registrations, registration.ID)
			Register[I, O](session, uri, options, procedure)
		}
		session.restores[registration.ID] = onRejoin

		return registration, nil
	}

	__logger.Error("during register", "error", e)
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
		delete(session.restores, subscriptionID)
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
		delete(session.restores, registrationID)
		session.logger.Debug("registration successfully dettached", logData)
	}
	session.logger.Error("during unregister", "error", e, logData)
	return e
}

func Leave(
	session *Session,
	reason string,
) {
	logData := slog.Group("leave", "reason", reason)
	session.logger.Debug("trying to leave", logData)
	session.peer.Close()
}
