package wamp

import (
	"errors"
	"strings"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

type MessageKind int8

const (
	MK_CALL    MessageKind = 127
	MK_CANCEL              = 126
	MK_NEXT                = 125
	MK_STOP                = 124
	MK_PUBLISH             = 1
	MK_ACCEPT              = 0
	MK_YIELD               = -125
	MK_ERROR               = -126
	MK_REPLY               = -127
)

type messageProto[F any] struct {
	id       string
	kind     MessageKind
	features F
	peer     *Peer
}

func (message *messageProto[F]) ID() string {
	return message.id
}

func (message *messageProto[F]) Kind() MessageKind {
	return message.kind
}

func (message *messageProto[F]) Features() F {
	return message.features
}

func (message *messageProto[F]) setPeer(instance *Peer) {
	message.peer = instance
}

func (message *messageProto[F]) getPeer() *Peer {
	return message.peer
}

type messagePayload interface {
	Content() any
	Payload(any) error
}

type messagePayloadField[T any] struct {
	value T
}

func (field *messagePayloadField[T]) Content() any {
	return field.value
}

func (field *messagePayloadField[T]) Payload(__v any) error {
	v, ok := __v.(*T)
	if ok {
		*v = field.value
		return nil
	}
	return errors.New("InvalidPayload")
}

type messageRouteField[T any] struct {
	route T
}

func (field *messageRouteField[T]) Route() T {
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
	return &messageProto[*AcceptFeatures]{id, MK_ACCEPT, features, nil}
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
	messagePayload
	Route() *PublishRoute
}

func MakePublishEvent(
	id string,
	features *PublishFeatures,
	data messagePayload,
	route *PublishRoute,
) PublishEvent {
	type message struct {
		*messageProto[*PublishFeatures]
		*messageRouteField[*PublishRoute]
		messagePayload
	}
	return &message{
		&messageProto[*PublishFeatures]{id, MK_PUBLISH, features, nil},
		&messageRouteField[*PublishRoute]{route},
		data,
	}
}

func newPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(
		wampShared.NewID(),
		features,
		&messagePayloadField[T]{data},
		new(PublishRoute),
	)
}

type CallFeatures struct {
	URI     string        `json:"URI"`
	Timeout time.Duration `json:"timeout"`
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
	messagePayload
	Route() *CallRoute
}

func MakeCallEvent(
	id string,
	features *CallFeatures,
	data messagePayload,
	route *CallRoute,
) CallEvent {
	type message struct {
		*messageProto[*CallFeatures]
		*messageRouteField[*CallRoute]
		messagePayload
	}
	return &message{
		&messageProto[*CallFeatures]{id, MK_CALL, features, nil},
		&messageRouteField[*CallRoute]{route},
		data,
	}
}

func newCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(
		wampShared.NewID(),
		features,
		&messagePayloadField[T]{data},
		new(CallRoute),
	)
}

type ReplyFeatures struct {
	InvocationID   string   `json:"invocationID"`
	VisitedRouters []string `json:"visitedRouters"`
}

type ReplyEvent interface {
	Event
	Features() *ReplyFeatures
	messagePayload
}

func MakeReplyEvent(
	id string,
	kind MessageKind,
	features *ReplyFeatures,
	data messagePayload,
) ReplyEvent {
	type message struct {
		*messageProto[*ReplyFeatures]
		messagePayload
	}
	return &message{&messageProto[*ReplyFeatures]{id, kind, features, nil}, data}
}

func NewReplyEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_REPLY,
		&ReplyFeatures{source.ID(), []string{}},
		&messagePayloadField[T]{data},
	)
}

type errorEventPayload struct {
	Code string `json:"code"`
}

func NewErrorEvent(source Event, e error) ReplyEvent {
	errorMessage := e.Error()
	payload := errorEventPayload{errorMessage}
	data := messagePayloadField[errorEventPayload]{payload}
	return MakeReplyEvent(wampShared.NewID(), MK_ERROR, &ReplyFeatures{source.ID(), []string{}}, &data)
}

type YieldEvent = ReplyEvent

func newYieldEvent[T any](source Event, data T) YieldEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_YIELD,
		&ReplyFeatures{source.ID(), []string{}},
		&messagePayloadField[T]{data},
	)
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
		*messageProto[*ReplyFeatures]
	}
	return &message{&messageProto[*ReplyFeatures]{id, MK_CANCEL, features, nil}}
}

func newCancelEvent(source Event) CancelEvent {
	return MakeCancelEvent(wampShared.NewID(), &ReplyFeatures{source.ID(), []string{}})
}

type NextFeatures struct {
	YieldID string `json:"yieldID"`
}

type NextEvent interface {
	Event
	Features() *NextFeatures
}

func MakeNextEvent(id string, features *NextFeatures) NextEvent {
	type message struct {
		*messageProto[*NextFeatures]
	}
	return &message{&messageProto[*NextFeatures]{id, MK_NEXT, features, nil}}
}

func newNextEvent(source Event) NextEvent {
	return MakeNextEvent(
		wampShared.NewID(),
		&NextFeatures{source.ID()},
	)
}

type Resource[T any] struct {
	ID       string `json:"ID"`
	URI      string `json:"URI"`
	AuthorID string `json:"authorID"`
	Options  T      `json:"options"`
}

func (resource *Resource[T]) Native() bool {
	return strings.HasPrefix(resource.URI, "wamp.")
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

type PublishEndpoint func(PublishEvent)

type CallEndpoint func(CallEvent) ReplyEvent
