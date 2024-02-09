package wamp

import (
	"slices"
	"strings"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

type MessageKind int8

const (
	MK_CALL      MessageKind = 127
	MK_CANCEL                = 126
	MK_SUBEVENT              = 9
	MK_PUBLISH               = 1
	MK_ACCEPT                = 0
	MK_UNDEFINED             = -1
	MK_ERROR                 = -126
	MK_REPLY                 = -127
)

type eventProto[F any] struct {
	id       string
	kind     MessageKind
	features F
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

type eventPayload interface {
	Payload() any
}

type payloadEventField[T any] struct {
	payload T
}

func (field *payloadEventField[T]) Payload() any {
	return field.payload
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
}

type AcceptFeatures struct {
	SourceID string `json:"sourceID"`
}

type AcceptEvent interface {
	Event
	Features() *AcceptFeatures
}

func MakeAcceptEvent(id string, features *AcceptFeatures) AcceptEvent {
	return &eventProto[*AcceptFeatures]{id, MK_ACCEPT, features}
}

func newAcceptEvent(source Event) AcceptEvent {
	features := AcceptFeatures{source.ID()}
	return MakeAcceptEvent(wampShared.NewID(), &features)
}

type PublishFeatures struct {
	URI                string   `json:"URI"`
	ExcludeSubscribers []string `json:"excludeSubscribers"`
	ExcludeRoles       []string `json:"excludeRoles"`
	IncludeSubscribers []string `json:"includeSubscribers"`
	IncludeRoles       []string `json:"includeRoles"`
}

func (features *PublishFeatures) Authorized(subscriberID string, role string) bool {
	if slices.Contains(features.ExcludeSubscribers, subscriberID) ||
		slices.Contains(features.ExcludeRoles, role) {
		return false
	}
	return (len(features.IncludeSubscribers) == 0 || slices.Contains(features.IncludeSubscribers, subscriberID)) &&
		(len(features.IncludeRoles) == 0 || slices.Contains(features.IncludeRoles, role))
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

func MakePublishEvent[T any](
	id string,
	features *PublishFeatures,
	data T,
	route *PublishRoute,
) PublishEvent {
	type message struct {
		*eventProto[*PublishFeatures]
		*payloadEventField[T]
		*routeEventField[*PublishRoute]
	}
	return &message{
		&eventProto[*PublishFeatures]{id, MK_PUBLISH, features},
		&payloadEventField[T]{data},
		&routeEventField[*PublishRoute]{route},
	}
}

func newPublishEvent[T any](features *PublishFeatures, data T) PublishEvent {
	return MakePublishEvent(
		wampShared.NewID(),
		features,
		data,
		new(PublishRoute),
	)
}

type CallFeatures struct {
	URI          string   `json:"URI"`
	Timeout      uint64   `json:"timeout"`
	ExcludeRoles []string `json:"excludeRoles"`
	IncludeRoles []string `json:"includeRoles"`
}

func (features *CallFeatures) Authorized(role string) bool {
	if slices.Contains(features.ExcludeRoles, role) {
		return false
	}
	return len(features.IncludeRoles) == 0 || slices.Contains(features.IncludeRoles, role)
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
	data any,
	route *CallRoute,
) CallEvent {
	type message struct {
		*eventProto[*CallFeatures]
		*routeEventField[*CallRoute]
		*payloadEventField[any]
	}
	return &message{
		&eventProto[*CallFeatures]{id, MK_CALL, features},
		&routeEventField[*CallRoute]{route},
		&payloadEventField[any]{data},
	}
}

func newCallEvent[T any](features *CallFeatures, data T) CallEvent {
	return MakeCallEvent(
		wampShared.NewID(),
		features,
		data,
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
	return &message{
		&eventProto[*ReplyFeatures]{id, MK_CANCEL, features},
	}
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
	data any,
) ReplyEvent {
	type message struct {
		*eventProto[*ReplyFeatures]
		*payloadEventField[any]
	}
	return &message{
		&eventProto[*ReplyFeatures]{id, kind, features},
		&payloadEventField[any]{data},
	}
}

func NewReplyEvent[T any](source Event, data T) ReplyEvent {
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_REPLY,
		&ReplyFeatures{source.ID(), []string{}},
		data,
	)
}

type errorEventPayload struct {
	Message string `json:"message"`
}

type ErrorEvent = ReplyEvent

func NewErrorEvent(source Event, e error) ErrorEvent {
	errorMessage := e.Error()
	return MakeReplyEvent(
		wampShared.NewID(),
		MK_ERROR,
		&ReplyFeatures{source.ID(), []string{}},
		errorEventPayload{errorMessage},
	)
}

type SubEvent interface {
	Event
	Features() string
	eventPayload
}

func MakeSubEvent(id string, tranceiver string, data any) SubEvent {
	type message struct {
		*eventProto[string]
		*payloadEventField[any]
	}
	return &message{
		&eventProto[string]{id, MK_SUBEVENT, tranceiver},
		&payloadEventField[any]{data},
	}
}

func newSubEvent[T any](tranceiver string, data T) SubEvent {
	return MakeSubEvent(wampShared.NewID(), tranceiver, data)
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
	Route        []string `json:"route"`
	ExcludeRoles []string `json:"excludeRoles"`
	IncludeRoles []string `json:"includeRoles"`
}

func (options *resourceOptions) Authorized(role string) bool {
	if slices.Contains(options.ExcludeRoles, role) {
		return false
	}
	return len(options.IncludeRoles) == 0 || slices.Contains(options.IncludeRoles, role)
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
