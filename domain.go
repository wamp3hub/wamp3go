package wamp

import (
	"errors"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

type MessageKind int8

const (
	MK_ACCEPT  MessageKind = 0
	MK_PUBLISH             = 1
	MK_CALL                = 127
	MK_NEXT                = 126
	MK_REPLY               = -127
	MK_YIELD               = -126
	MK_ERROR               = -125
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
	payload T
}

func (field *messagePayloadField[T]) Content() any {
	return field.payload
}

func (field *messagePayloadField[T]) Payload(__v any) error {
	v, ok := __v.(*T)
	if ok {
		*v = field.payload
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

func NewPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(
		wampShared.NewID(),
		features,
		&messagePayloadField[T]{data},
		new(PublishRoute),
	)
}

type CallFeatures struct {
	URI string `json:"URI"`
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

type callMessage struct {
	*messageProto[*CallFeatures]
	*messageRouteField[*CallRoute]
	messagePayload
}

func MakeCallEvent(
	id string,
	features *CallFeatures,
	data messagePayload,
	route *CallRoute,
) CallEvent {
	return &callMessage{
		&messageProto[*CallFeatures]{id, MK_CALL, features, nil},
		&messageRouteField[*CallRoute]{route},
		data,
	}
}

func NewCallEvent[T any](features *CallFeatures, data T) CallEvent {
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

func newYieldEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_YIELD,
		&ReplyFeatures{source.ID(), []string{}},
		&messagePayloadField[T]{data},
	)
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
