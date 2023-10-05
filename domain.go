package wamp3go

import (
	"errors"

	"github.com/rs/xid"
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

func (message *messageProto[F]) bind(instance *Peer) {
	message.peer = instance
}

func (message *messageProto[F]) Peer() *Peer {
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
	bind(*Peer)
	Peer() *Peer
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
	return MakeAcceptEvent(xid.New().String(), &features)
}

type PublishFeatures struct {
	URI     string
	Include []string
	Exclude []string
}

type PublishRoute struct {
	PublisherID  string `json:"publisherID"`
	SubscriberID string `json:"subscriberID"`
	EndpointID   string `json:"endpointID"`
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
		xid.New().String(),
		features,
		&messagePayloadField[T]{data},
		new(PublishRoute),
	)
}

type CallFeatures struct {
	URI string `json:"uri"`
}

type CallRoute struct {
	CallerID   string `json:"callerID"`
	ExecutorID string `json:"executorID"`
	EndpointID string `json:"endpointID"`
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

func NewCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(
		xid.New().String(),
		features,
		&messagePayloadField[T]{data},
		new(CallRoute),
	)
}

type ReplyFeatures struct {
	InvocationID string `json:"invocationID"`
}

type ReplyEvent interface {
	Event
	Features() *ReplyFeatures
	Error() error
	Done() bool
	messagePayload
}

type replyMessage struct {
	*messageProto[*ReplyFeatures]
	messagePayload
}

func (message *replyMessage) Done() bool {
	return message.Kind() != MK_YIELD
}

func (message *replyMessage) Error() error {
	if message.Kind() != MK_ERROR {
		return nil
	}

	payload := new(ErrorEventPayload)
	e := message.Payload(payload)
	if e == nil {
		return errors.New(payload.Code)
	}
	return e
}

func MakeReplyEvent(
	id string,
	kind MessageKind,
	features *ReplyFeatures,
	data messagePayload,
) ReplyEvent {
	return &replyMessage{&messageProto[*ReplyFeatures]{id, kind, features, nil}, data}
}

func NewReplyEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		xid.New().String(),
		MK_REPLY,
		&ReplyFeatures{source.ID()},
		&messagePayloadField[T]{data},
	)
}

type ErrorEventPayload struct {
	Code string `json:"code"`
}

func NewErrorEvent(source Event, e error) ReplyEvent {
	errorMessage := e.Error()
	payload := ErrorEventPayload{errorMessage}
	data := messagePayloadField[ErrorEventPayload]{payload}
	return MakeReplyEvent(xid.New().String(), MK_ERROR, &ReplyFeatures{source.ID()}, &data)
}

func newYieldEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		xid.New().String(),
		MK_YIELD,
		&ReplyFeatures{source.ID()},
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

func newNextEvent(generatorID string) NextEvent {
	return MakeNextEvent(xid.New().String(), &NextFeatures{generatorID})
}

type SubscribeOptions struct{}

type RegisterOptions struct{}

type Resource[T any] struct {
	ID       string
	URI      string
	AuthorID string
	Options  T
}

type Subscription = Resource[*SubscribeOptions]

type Registration = Resource[*RegisterOptions]

type PublishEndpoint func(PublishEvent)

type CallEndpoint func(CallEvent) ReplyEvent
