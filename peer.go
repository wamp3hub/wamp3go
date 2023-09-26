package wamp3go

import (
	"errors"
	"log"

	"wamp3go/shared"
)

type Event any

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
		publishEventProducer,
		publishEventConsumer,
		callEventProducer,
		callEventConsumer,
	}
}

func (peer *Peer) Consume() {
	q := make(QEvent)
	go peer.Transport.Receive(q)
	for event := range q {
		e := error(nil)
		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.PendingAcceptEvents.Throw(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = peer.PendingReplyEvents.Throw(features.InvocationID, event)
		case PublishEvent:
			peer.publishEventProducer.Produce(event)
		case CallEvent:
			peer.callEventProducer.Produce(event)
		default:
			e = errors.New("InvalidEvent")
		}
		if e != nil {
			log.Printf("peer %s", e)
		}
	}
	peer.publishEventProducer.Close()
	peer.callEventProducer.Close()
}
