package wamp3go

import (
	"errors"
	"log"

	"github.com/wamp3hub/wamp3go/shared"
)

type Event interface {
	ID() string
	Kind() MessageKind
}

type QEvent chan Event

type Serializer interface {
	Code() string
	Encode(Event) ([]byte, error)
	Decode([]byte) (Event, error)
}

type Transport interface {
	Send(Event) error
	Receive(QEvent)
	Close() error
}

type Peer struct {
	ID                    string
	Transport             Transport
	PendingAcceptEvents   shared.PendingMap[AcceptEvent]
	PendingReplyEvents    shared.PendingMap[ReplyEvent]
	PendingNextEvents     shared.PendingMap[NextEvent]
	publishEventProducer  *shared.Producer[PublishEvent]
	IncomingPublishEvents *shared.Consumer[PublishEvent]
	callEventProducer     *shared.Producer[CallEvent]
	IncomingCallEvents    *shared.Consumer[CallEvent]
}

func NewPeer(ID string, transport Transport) *Peer {
	publishEventProducer, publishEventConsumer := shared.NewStream[PublishEvent]()
	callEventProducer, callEventConsumer := shared.NewStream[CallEvent]()
	return &Peer{
		ID,
		transport,
		shared.NewPendingMap[AcceptEvent](),
		shared.NewPendingMap[ReplyEvent](),
		shared.NewPendingMap[NextEvent](),
		publishEventProducer,
		publishEventConsumer,
		callEventProducer,
		callEventConsumer,
	}
}

func (peer *Peer) Send(event Event) error {
	acceptEventPromise := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	peer.Transport.Send(event)
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return errors.New("TimedOut")
}

func (peer *Peer) Consume() {
	q := make(QEvent)
	go peer.Transport.Receive(q)
	for event := range q {
		e := error(nil)
		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			peer.PendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = peer.PendingReplyEvents.Complete(features.InvocationID, event)
			if e == nil {
				acceptEvent := NewAcceptEvent(event.ID())
				peer.Transport.Send(acceptEvent)
			} else {
				log.Printf("[peer] %s (ID=%s event=%s)", e, peer.ID, event)
			}
		case NextEvent:
			features := event.Features()
			e = peer.PendingNextEvents.Complete(features.GeneratorID, event)
			if e == nil {
				acceptEvent := NewAcceptEvent(event.ID())
				peer.Transport.Send(acceptEvent)
			} else {
				log.Printf("[peer] %s (ID=%s event=%s)", e, peer.ID, event)
			}
		case PublishEvent:
			peer.publishEventProducer.Produce(event)
			acceptEvent := NewAcceptEvent(event.ID())
			peer.Transport.Send(acceptEvent)
		case CallEvent:
			peer.callEventProducer.Produce(event)
			acceptEvent := NewAcceptEvent(event.ID())
			peer.Transport.Send(acceptEvent)
		default:
			log.Printf("[peer] InvalidEvent (ID=%s event=%s)", peer.ID, event)
		}
	}
	peer.publishEventProducer.Close()
	peer.callEventProducer.Close()
}
