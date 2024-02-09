package wamp

import (
	"errors"
)

var (
	ErrorGeneratorExit = errors.New("generator exit")
)

type generatorExitException struct {
	Source Event
}

func (generatorExitException) Error() string {
	return "GeneratorExit"
}

func GeneratorExit(source Event) *generatorExitException {
	return &generatorExitException{source}
}

// type NewGeneratorPayload struct {
// 	ID string `json:"id"`
// }

// type remoteGenerator[T any] struct {
// 	done        bool
// 	ID          string
// 	lastYieldID string
// 	peer        *Peer
// 	logger      *slog.Logger
// }

// func (generator *remoteGenerator[T]) Active() bool {
// 	return !generator.done
// }

// func NewRemoteGenerator[T any](
// 	session *Session,
// 	initialEvent YieldEvent,
// ) *remoteGenerator[T] {
// 	payload, _ := ReadPayload[NewGeneratorPayload](initialEvent)
// 	logData := slog.Group(
// 		"generator",
// 		"ID", payload.ID,
// 	)
// 	return &remoteGenerator[T]{
// 		false,
// 		payload.ID,
// 		initialEvent.ID(),
// 		session.peer,
// 		session.logger.With("name", "RemoteGenerator", logData),
// 	}
// }

// func CallGenerator[O, I any](
// 	session *Session,
// 	features *CallFeatures,
// 	inPayload I,
// ) (*remoteGenerator[O], error) {
// 	logData := slog.Group(
// 		"generator",
// 		"URI", features.URI,
// 	)
// 	session.logger.Debug("trying to initialize remote generator", logData)

// 	pendingResponse := Call[any](session, features, inPayload)
// 	yieldEvent, _, e := pendingResponse.Await()
// 	if e == nil {
// 		session.logger.Debug("remote generator successfully initialized", logData)
// 		generator := NewRemoteGenerator[O](session, yieldEvent)
// 		return generator, nil
// 	}

// 	session.logger.Error("during initialize remote generator", "error", e, logData)
// 	return nil, e
// }

// func (generator *remoteGenerator[T]) Next(
// 	timeout uint64,
// ) (response ReplyEvent, outPayload T, e error) {
// 	if generator.done {
// 		generator.logger.Error("generator already closed")
// 		return nil, outPayload, ErrorGeneratorExit
// 	}

// 	logData := slog.Group(
// 		"nextEvent",
// 		"yieldID", generator.lastYieldID,
// 		"timeout", timeout,
// 	)
// 	generator.logger.Debug("trying to get next", logData)

// 	nextFeatures := Subevent{generator.ID, generator.lastYieldID, timeout}
// 	nextEvent := newNextEvent(&nextFeatures)
// 	responseTimeout := time.Duration(2*timeout) * time.Second
// 	responsePromise, cancelResponsePromise := generator.peer.PendingReplyEvents.New(
// 		nextEvent.ID(),
// 		responseTimeout,
// 	)
// 	pendingResponse := newPendingResponse[T](responsePromise, cancelResponsePromise)
// 	ok := generator.peer.Send(nextEvent, DEFAULT_RESEND_COUNT)
// 	if !ok {
// 		generator.logger.Error("next event dispatch error", logData)
// 		cancelResponsePromise()
// 		generator.done = true
// 		return nil, outPayload, ErrorDispatch
// 	}

// 	response, outPayload, e = pendingResponse.Await()
// 	if e == nil && response.Kind() == MK_YIELD {
// 		generator.logger.Debug("yield event successfully received", logData)
// 		generator.lastYieldID = response.ID()
// 	} else {
// 		generator.logger.Debug("destroying generator", "error", e, logData)
// 		generator.done = true
// 	}

// 	return response, outPayload, e
// }

// func (generator *remoteGenerator[T]) Stop() error {
// 	if generator.done {
// 		generator.logger.Error("generator already closed")
// 		return ErrorGeneratorExit
// 	}

// 	generator.logger.Debug("trying to stop generator")

// 	stopEvent := NewStopEvent(generator.ID)
// 	ok := generator.peer.Send(stopEvent, DEFAULT_RESEND_COUNT)
// 	if ok {
// 		generator.done = true
// 		generator.logger.Debug("generator successfully stopped")
// 		return nil
// 	}

// 	generator.logger.Error("generator stop event dispatch error")
// 	return ErrorDispatch
// }

// func yieldNext(
// 	peer *Peer,
// 	generatorID string,
// 	lifetime time.Duration,
// 	yieldEvent YieldEvent,
// 	__logger *slog.Logger,
// ) NextEvent {
// 	logger := __logger.With(
// 		slog.Group(
// 			"yieldEvent",
// 			"ID", yieldEvent.ID(),
// 			"GeneratorID", generatorID,
// 			"GeneratorLifetime", lifetime,
// 		),
// 	)

// 	nextEventPromise, cancelNextEventPromise := peer.PendingNextEvents.New(yieldEvent.ID(), 0)

// 	stopEventPromise, cancelStopEventPromise := peer.PendingCancelEvents.New(generatorID, lifetime)

// 	logger.Debug("trying to send yield event")
// 	ok := peer.Send(yieldEvent, DEFAULT_RESEND_COUNT)
// 	if !ok {
// 		logger.Error("yield event dispatch error (destroying generator)")
// 		cancelNextEventPromise()
// 		cancelStopEventPromise()
// 		panic("protocol error")
// 	}

// 	select {
// 	case _, done := <-stopEventPromise:
// 		if done {
// 			logger.Warn("generator stop event received (destroying generator)")
// 		} else {
// 			logger.Warn("generator lifetime expired (destroying generator)")
// 		}
// 		cancelNextEventPromise()
// 		panic("generator destroy")
// 	case nextEvent := <-nextEventPromise:
// 		cancelStopEventPromise()
// 		logger.Debug("generator next", "nextEvent.ID", nextEvent.ID())
// 		return nextEvent
// 	}
// }

// func Yield[I any](
// 	source Event,
// 	inPayload I,
// ) NextEvent {
// 	router := source.getPeer()
// 	__logger := router.logger.With(
// 		"name", "Yield",
// 		"sourceEvent.Kind", source.Kind(),
// 	)

// 	lifetime := DEFAULT_GENERATOR_LIFETIME * time.Second

// 	callEvent, ok := source.(CallEvent)
// 	if ok {
// 		generator := NewGeneratorPayload{wampShared.NewID()}
// 		yieldEvent := newYieldEvent(callEvent, generator)
// 		source = yieldNext(router, generator.ID, lifetime, yieldEvent, __logger)
// 	}

// 	nextEvent, ok := source.(NextEvent)
// 	if ok {
// 		nextFeatures := nextEvent.Features()
// 		yieldEvent := newYieldEvent(nextEvent, inPayload)
// 		return yieldNext(router, nextFeatures.GeneratorID, lifetime, yieldEvent, __logger)
// 	}

// 	__logger.Error("invalid source event (destroying generator)")
// 	panic("invalid source event")
// }
