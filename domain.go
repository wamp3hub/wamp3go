package wamp3go

import (
	"errors"

	"github.com/google/uuid"
)

type MessageKind int8

const (
	MK_PUBLISH MessageKind = 1
	MK_ACCEPT              = -1
	MK_CALL                = 127
	MK_REPLY               = -127
)

type messageProto[F any] struct {
	id       string
	kind     MessageKind
	features F
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
	ID() string
	Kind() MessageKind
	Features() *PublishFeatures
	messagePayload
	Route() *PublishRoute
}

func MakePublishEvent(id string, features *PublishFeatures, data messagePayload, route *PublishRoute) PublishEvent {
	type message struct {
		*messageProto[*PublishFeatures]
		*messageRouteField[*PublishRoute]
		messagePayload
	}
	return &message{
		&messageProto[*PublishFeatures]{id, MK_PUBLISH, features},
		&messageRouteField[*PublishRoute]{route},
		data,
	}
}

func NewPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(uuid.NewString(), features, &messagePayloadField[T]{data}, new(PublishRoute))
}

type AcceptFeatures struct {
	SourceID string `json:"sourceID"`
}

type AcceptEvent interface {
	ID() string
	Kind() MessageKind
	Features() *AcceptFeatures
}

func MakeAcceptEvent(id string, features *AcceptFeatures) AcceptEvent {
	return &messageProto[*AcceptFeatures]{id, MK_ACCEPT, features}
}

func NewAcceptEvent(sourceID string) AcceptEvent {
	features := AcceptFeatures{sourceID}
	return MakeAcceptEvent(uuid.NewString(), &features)
}

type CallFeatures struct {
	URI string
}

type CallRoute struct {
	CallerID   string `json:"callerID"`
	ExecutorID string `json:"executorID"`
	EndpointID string `json:"endpointID"`
}

type CallEvent interface {
	ID() string
	Kind() MessageKind
	Features() *CallFeatures
	messagePayload
	Route() *CallRoute
}

func MakeCallEvent(id string, features *CallFeatures, data messagePayload, route *CallRoute) CallEvent {
	type message struct {
		*messageProto[*CallFeatures]
		*messageRouteField[*CallRoute]
		messagePayload
	}
	return &message{
		&messageProto[*CallFeatures]{id, MK_CALL, features},
		&messageRouteField[*CallRoute]{route},
		data,
	}
}

func NewCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(uuid.NewString(), features, &messagePayloadField[T]{data}, new(CallRoute))
}

type ReplyFeatures struct {
	OK           bool   `json:"OK"`
	InvocationID string `json:"invocationID"`
}

type ReplyEvent interface {
	ID() string
	Kind() MessageKind
	Features() *ReplyFeatures
	messagePayload
}

func MakeReplyEvent(id string, features *ReplyFeatures, data messagePayload) ReplyEvent {
	type message struct {
		*messageProto[*ReplyFeatures]
		messagePayload
	}
	return &message{&messageProto[*ReplyFeatures]{id, MK_REPLY, features}, data}
}

func NewReplyEvent[T any](invocationID string, data T) ReplyEvent {
	return MakeReplyEvent(uuid.NewString(), &ReplyFeatures{true, invocationID}, &messagePayloadField[T]{data})
}

type ErrorPayload struct {
	Code string `json:"code"`
}

func NewErrorEvent(invocationID string, e error) ReplyEvent {
	errorMessage := e.Error()
	payload := ErrorPayload{errorMessage}
	data := messagePayloadField[ErrorPayload]{payload}
	return MakeReplyEvent(uuid.NewString(), &ReplyFeatures{false, invocationID}, &data)
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

type publishEndpoint func(PublishEvent)

type callEndpoint func(CallEvent) ReplyEvent
