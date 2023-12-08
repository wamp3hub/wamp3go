package wamp

import (
	"errors"
	"strings"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var (
	ErrorCancelled = errors.New("Cancelled")
	ExitGenerator = errors.New("ExitGenerator")
	InvalidPayload = errors.New("InvalidPayload")
	ProcedureNotFound = errors.New("ProcedureNotFound")
)

type MessageKind int8

const (
	MK_CALL      MessageKind = 127
	MK_CANCEL                = 126
	MK_NEXT                  = 125
	MK_PUBLISH               = 1
	MK_ACCEPT                = 0
	MK_UNDEFINED             = -1
	MK_YIELD                 = -125
	MK_ERROR                 = -126
	MK_REPLY                 = -127
)

type eventProto[F any] struct {
	id       string
	kind     MessageKind
	features F
	peer     *Peer
}

func (event *eventProto[F]) ID() string {
	return event.id
}

func (event *eventProto[F]) Kind() MessageKind {
	return event.kind
}

func (event *eventProto[F]) Features() F {
	return event.features
}

func (event *eventProto[F]) setPeer(instance *Peer) {
	event.peer = instance
}

func (event *eventProto[F]) getPeer() *Peer {
	return event.peer
}

type eventPayload interface {
	Content() any
	Payload(any) error
}

type payloadEventField[T any] struct {
	value T
}

func (field *payloadEventField[T]) Content() any {
	return field.value
}

func (field *payloadEventField[T]) Payload(__v any) error {
	v, ok := __v.(*T)
	if ok {
		*v = field.value
		return nil
	}
	return InvalidPayload
}

type routeEventField[T any] struct {
	route T
}

func (field *routeEventField[T]) Route() T {
	return field.route
}

type Event interface {
	ID() string
	Kind() MessageKind
	setPeer(*Peer)
	getPeer() *Peer
}

type AcceptFeatures struct {
	SourceID string `json:"sourceID"`
}

type AcceptEvent interface {
	Event
	Features() *AcceptFeatures
}

func MakeAcceptEvent(id string, features *AcceptFeatures) AcceptEvent {
	return &eventProto[*AcceptFeatures]{id, MK_ACCEPT, features, nil}
}

func newAcceptEvent(source Event) AcceptEvent {
	features := AcceptFeatures{source.ID()}
	return MakeAcceptEvent(wampShared.NewID(), &features)
}

type PublishFeatures struct {
	URI     string   `json:"URI"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type PublishRoute struct {
	PublisherID    string   `json:"publisherID"`
	SubscriberID   string   `json:"subscriberID"`
	EndpointID     string   `json:"endpointID"`
	VisitedRouters []string `json:"visitedRouters"`
}

type PublishEvent interface {
	Event
	Features() *PublishFeatures
	eventPayload
	Route() *PublishRoute
}

func MakePublishEvent(
	id string,
	features *PublishFeatures,
	data eventPayload,
	route *PublishRoute,
) PublishEvent {
	type message struct {
		*eventProto[*PublishFeatures]
		*routeEventField[*PublishRoute]
		eventPayload
	}
	return &message{
		&eventProto[*PublishFeatures]{id, MK_PUBLISH, features, nil},
		&routeEventField[*PublishRoute]{route},
		data,
	}
}

func newPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(
		wampShared.NewID(),
		features,
		&payloadEventField[T]{data},
		new(PublishRoute),
	)
}

type CallFeatures struct {
	URI     string `json:"URI"`
	Timeout uint64 `json:"timeout"`
}

type CallRoute struct {
	CallerID       string   `json:"callerID"`
	ExecutorID     string   `json:"executorID"`
	EndpointID     string   `json:"endpointID"`
	VisitedRouters []string `json:"visitedRouters"`
}

type CallEvent interface {
	Event
	Features() *CallFeatures
	eventPayload
	Route() *CallRoute
}

func MakeCallEvent(
	id string,
	features *CallFeatures,
	data eventPayload,
	route *CallRoute,
) CallEvent {
	type message struct {
		*eventProto[*CallFeatures]
		*routeEventField[*CallRoute]
		eventPayload
	}
	return &message{
		&eventProto[*CallFeatures]{id, MK_CALL, features, nil},
		&routeEventField[*CallRoute]{route},
		data,
	}
}

func newCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(
		wampShared.NewID(),
		features,
		&payloadEventField[T]{data},
		new(CallRoute),
	)
}

type ReplyFeatures struct {
	InvocationID   string   `json:"invocationID"`
	VisitedRouters []string `json:"visitedRouters"`
}

type CancelEvent interface {
	Event
	Features() *ReplyFeatures
}

func MakeCancelEvent(
	id string,
	features *ReplyFeatures,
) CancelEvent {
	type message struct {
		*eventProto[*ReplyFeatures]
	}
	return &message{&eventProto[*ReplyFeatures]{id, MK_CANCEL, features, nil}}
}

func newCancelEvent(source Event) CancelEvent {
	return MakeCancelEvent(wampShared.NewID(), &ReplyFeatures{source.ID(), []string{}})
}

type ReplyEvent interface {
	Event
	Features() *ReplyFeatures
	eventPayload
}

func MakeReplyEvent(
	id string,
	kind MessageKind,
	features *ReplyFeatures,
	data eventPayload,
) ReplyEvent {
	type message struct {
		*eventProto[*ReplyFeatures]
		eventPayload
	}
	return &message{&eventProto[*ReplyFeatures]{id, kind, features, nil}, data}
}

func NewReplyEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_REPLY,
		&ReplyFeatures{source.ID(), []string{}},
		&payloadEventField[T]{data},
	)
}

type errorEventPayload struct {
	Message string `json:"message"`
}

type ErrorEvent = ReplyEvent

func NewErrorEvent(source Event, e error) ErrorEvent {
	errorMessage := e.Error()
	payload := errorEventPayload{errorMessage}
	data := payloadEventField[errorEventPayload]{payload}
	return MakeReplyEvent(wampShared.NewID(), MK_ERROR, &ReplyFeatures{source.ID(), []string{}}, &data)
}

type YieldEvent = ReplyEvent

func newYieldEvent[T any](source Event, data T) YieldEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_YIELD,
		&ReplyFeatures{source.ID(), []string{}},
		&payloadEventField[T]{data},
	)
}

type StopEvent = CancelEvent

func NewStopEvent(generatorID string) StopEvent {
	return MakeCancelEvent(wampShared.NewID(), &ReplyFeatures{generatorID, []string{}})
}

func GeneratorExit(source Event) ErrorEvent {
	return NewErrorEvent(source, ExitGenerator)
}

type NextFeatures struct {
	GeneratorID string `json:"generatorID"`
	YieldID     string `json:"yieldID"`
	Timeout     uint64 `json:"timeout"`
}

type NextEvent interface {
	Event
	Features() *NextFeatures
}

func MakeNextEvent(id string, features *NextFeatures) NextEvent {
	type message struct {
		*eventProto[*NextFeatures]
	}
	return &message{&eventProto[*NextFeatures]{id, MK_NEXT, features, nil}}
}

func newNextEvent(features *NextFeatures) NextEvent {
	return MakeNextEvent(wampShared.NewID(), features)
}

type Resource[T any] struct {
	ID       string `json:"ID"`
	URI      string `json:"URI"`
	AuthorID string `json:"authorID"`
	Options  T      `json:"options"`
}

func (resource *Resource[T]) Native() bool {
	return strings.HasPrefix(resource.URI, "wamp.router.")
}

type resourceOptions struct {
	Route []string `json:"route"`
}

func (options *resourceOptions) Entrypoint() string {
	return options.Route[0]
}

func (options *resourceOptions) Distance() int {
	return len(options.Route)
}

type SubscribeOptions = resourceOptions

type RegisterOptions = resourceOptions

type Subscription = Resource[*SubscribeOptions]

type Registration = Resource[*RegisterOptions]
