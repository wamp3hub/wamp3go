package wamp3go

import (
	"errors"
	"log"
)

type Event any

type QEvent chan Event

type Serializer interface {
	Encode(Event) ([]byte, error)
	Decode([]byte) (Event, error)
}

type Transport interface {
	Send(Event) error
	Receive(QEvent)
	Close() error
}

type Peer struct {
	id            string
	Transport     Transport
	PublishEvents *Stream[PublishEvent]
	CallEvents    *Stream[CallEvent]
	AcceptEvents  PendingMap[AcceptEvent]
	ReplyEvents   PendingMap[ReplyEvent]
}

func (peer *Peer) ID() string {
	return peer.id
}

func NewPeer(ID string, transport Transport) *Peer {
	return &Peer{
		ID,
		transport,
		NewStream[PublishEvent](),
		NewStream[CallEvent](),
		make(PendingMap[AcceptEvent]),
		make(PendingMap[ReplyEvent]),
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
			e = peer.AcceptEvents.Throw(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = peer.ReplyEvents.Throw(features.InvocationID, event)
		case PublishEvent:
			peer.PublishEvents.Produce(event)
		case CallEvent:
			peer.CallEvents.Produce(event)
		default:
			e = errors.New("InvalidEvent")
		}
		if e != nil {
			log.Printf("peer %s", e)
		}
	}
	peer.PublishEvents.Reset()
	peer.CallEvents.Reset()
}
