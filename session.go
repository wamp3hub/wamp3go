package wamp3go

import (
	"errors"
	"log"
)

func ExtractError(event ReplyEvent) error {
	payload := new(ErrorPayload)
	e := event.Payload(payload)
	if e == nil {
		return errors.New(payload.Code)
	}
	return e
}

type Session struct {
	*Peer
	Subscriptions map[string]publishEndpoint
	Registrations map[string]callEndpoint
}

func NewSession(peer *Peer) *Session {
	return &Session{
		peer, make(map[string]publishEndpoint), make(map[string]callEndpoint),
	}
}

func (session *Session) Publish(event PublishEvent) error {
	e := session.Transport.Send(event)
	if e == nil {
		_, e = session.AcceptEvents.Catch(event.ID(), DEFAULT_TIMEOUT)
	}
	return e
}

func (session *Session) Call(event CallEvent) (response ReplyEvent) {
	e := session.Transport.Send(event)
	if e == nil {
		response, e = session.ReplyEvents.Catch(event.ID(), DEFAULT_TIMEOUT)
	}
	if e != nil {
		response = NewErrorEvent(event.ID(), e)
	}
	return response
}

type NewResourcePayload[O any] struct {
	URI     string
	Options *O
}

func (session *Session) Subscribe(
	uri string,
	features *SubscribeOptions,
	endpoint publishEndpoint,
) (*Subscription, error) {
	request := NewCallEvent(
		&CallFeatures{"wamp.subscribe"},
		&NewResourcePayload[SubscribeOptions]{uri, features},
	)
	response := session.Call(request)
	replyFeatures := response.Features()
	if replyFeatures.OK {
		subscription := new(Subscription)
		e := response.Payload(subscription)
		if e == nil {
			session.Subscriptions[subscription.ID] = endpoint
			return subscription, nil
		}
	}
	return nil, ExtractError(response)
}

func (session *Session) Register(
	uri string,
	features *RegisterOptions,
	endpoint callEndpoint,
) (*Registration, error) {
	request := NewCallEvent(
		&CallFeatures{"wamp.register"},
		&NewResourcePayload[RegisterOptions]{uri, features},
	)
	response := session.Call(request)
	replyFeatures := response.Features()
	if replyFeatures.OK {
		registration := new(Registration)
		e := response.Payload(registration)
		if e == nil {
			session.Registrations[registration.ID] = endpoint
			return registration, nil
		}
	}
	return nil, ExtractError(response)
}

type DeleteResourcePayload struct {
	ID string
}

func (session *Session) Unsubscribe(subscriptionID string) error {
	request := NewCallEvent(
		&CallFeatures{"wamp.unsubscribe"},
		&DeleteResourcePayload{subscriptionID},
	)
	response := session.Call(request)
	replyFeatures := response.Features()
	if replyFeatures.OK {
		return nil
	}
	return ExtractError(response)
}

func (session *Session) Unregister(registrationID string) error {
	request := NewCallEvent(
		&CallFeatures{"wamp.unregister"},
		&DeleteResourcePayload{registrationID},
	)
	response := session.Call(request)
	replyFeatures := response.Features()
	if replyFeatures.OK {
		return nil
	}
	return ExtractError(response)
}

// RENAME
func Initialize(session *Session) error {
	session.PublishEvents.Consume(
		func(request PublishEvent) {
			acceptEvent := NewAcceptEvent(request.ID())
			e := session.Transport.Send(acceptEvent)
			if e != nil {
				log.Printf("accept not sent (session.ID=%s) %s", session.ID(), e)
			}
			route := request.Route()
			endpoint, exist := session.Subscriptions[route.EndpointID]
			if exist {
				endpoint(request)
			} else {
				log.Printf(
					"subscription not found (session.ID=%s route.ID=%s publisher.ID=%s)",
					session.ID(), route.EndpointID, route.PublisherID,
				)
			}
		},
		func () {},
		1,
	)

	session.CallEvents.Consume(
		func(request CallEvent) {
			route := request.Route()
			endpoint, exist := session.Registrations[route.EndpointID]
			if exist {
				response := endpoint(request)
				e := session.Transport.Send(response)
				if e != nil {
					log.Printf("reply not sent (session.ID=%s) %s", session.ID(), e)
				}
			} else {
				log.Printf(
					"registration not found (session.ID=%s route.ID=%s caller.ID=%s)",
					session.ID(), route.EndpointID, route.CallerID,
				)
			}
		},
		func () {},
		1,
	)

	go session.Consume()

	log.Printf("session joined (peer.ID=%s)", session.ID())
	return nil
}
