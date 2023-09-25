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

type MessageProto[F any] struct {
	id       string
	kind     MessageKind
	features F
}

func (message *MessageProto[F]) ID() string {
	return message.id
}

func (message *MessageProto[F]) Kind() MessageKind {
	return message.kind
}

func (message *MessageProto[F]) Features() F {
	return message.features
}

type MessagePayload interface {
	Content() any
	Payload(any) error
}

type MessagePayloadField[T any] struct {
	payload T
}

func (field *MessagePayloadField[T]) Content() any {
	return field.payload
}

func (field *MessagePayloadField[T]) Payload(__v any) error {
	v, ok := __v.(*T)
	if ok {
		*v = field.payload
		return nil
	}
	return errors.New("InvalidPayload")
}

type MessageRouteField[T any] struct {
	route T
}

func (field *MessageRouteField[T]) Route() T {
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
	MessagePayload
	Route() *PublishRoute
}

func MakePublishEvent(id string, features *PublishFeatures, data MessagePayload, route *PublishRoute) PublishEvent {
	type message struct {
		*MessageProto[*PublishFeatures]
		*MessageRouteField[*PublishRoute]
		MessagePayload
	}
	return &message{
		&MessageProto[*PublishFeatures]{id, MK_PUBLISH, features},
		&MessageRouteField[*PublishRoute]{route},
		data,
	}
}

func NewPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(uuid.NewString(), features, &MessagePayloadField[T]{data}, new(PublishRoute))
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
	return &MessageProto[*AcceptFeatures]{id, MK_ACCEPT, features}
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
	MessagePayload
	Route() *CallRoute
}

func MakeCallEvent(id string, features *CallFeatures, data MessagePayload, route *CallRoute) CallEvent {
	type message struct {
		*MessageProto[*CallFeatures]
		*MessageRouteField[*CallRoute]
		MessagePayload
	}
	return &message{
		&MessageProto[*CallFeatures]{id, MK_CALL, features},
		&MessageRouteField[*CallRoute]{route},
		data,
	}
}

func NewCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(uuid.NewString(), features, &MessagePayloadField[T]{data}, new(CallRoute))
}

type ReplyFeatures struct {
	OK           bool   `json:"OK"`
	InvocationID string `json:"invocationID"`
}

type ReplyEvent interface {
	ID() string
	Kind() MessageKind
	Features() *ReplyFeatures
	MessagePayload
}

func MakeReplyEvent(id string, features *ReplyFeatures, data MessagePayload) ReplyEvent {
	type message struct {
		*MessageProto[*ReplyFeatures]
		MessagePayload
	}
	return &message{&MessageProto[*ReplyFeatures]{id, MK_REPLY, features}, data}
}

func NewReplyEvent[T any](invocationID string, data T) ReplyEvent {
	return MakeReplyEvent(uuid.NewString(), &ReplyFeatures{true, invocationID}, &MessagePayloadField[T]{data})
}

type ErrorPayload struct {
	Code string `json:"code"`
}

func NewErrorEvent(invocationID string, e error) ReplyEvent {
	errorMessage := e.Error()
	payload := ErrorPayload{errorMessage}
	data := MessagePayloadField[ErrorPayload]{payload}
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
