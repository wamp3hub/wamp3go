package wamp3go

import (
	"errors"
	"log"
	"sync"

	"github.com/wamp3hub/wamp3go/shared"
)

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
	ID                           string
	writeMutex                   *sync.Mutex
	Transport                    Transport
	PendingAcceptEvents          shared.PendingMap[AcceptEvent]
	PendingReplyEvents           shared.PendingMap[ReplyEvent]
	PendingNextEvents            shared.PendingMap[NextEvent]
	producePublishEvent          shared.Producible[PublishEvent]
	ConsumeIncomingPublishEvents shared.Consumable[PublishEvent]
	closePublishEvents           shared.Closeable
	produceCallEvent             shared.Producible[CallEvent]
	ConsumeIncomingCallEvents    shared.Consumable[CallEvent]
	closeCallEvents              shared.Closeable
}

func NewPeer(ID string, transport Transport) *Peer {
	consumePublishEvents, producePublishEvent, closePublishEvents := shared.NewStream[PublishEvent]()
	consumeCallEvents, produceCallEvent, closeCallEvents := shared.NewStream[CallEvent]()
	return &Peer{
		ID,
		new(sync.Mutex),
		transport,
		shared.NewPendingMap[AcceptEvent](),
		shared.NewPendingMap[ReplyEvent](),
		shared.NewPendingMap[NextEvent](),
		producePublishEvent,
		consumePublishEvents,
		closePublishEvents,
		produceCallEvent,
		consumeCallEvents,
		closeCallEvents,
	}
}

func (peer *Peer) Send(event Event) error {
	acceptEventPromise := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	// avoid concurrent write
	peer.writeMutex.Lock()
	peer.Transport.Send(event)
	peer.writeMutex.Unlock()
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return errors.New("TimedOut")
}

func (peer *Peer) Consume() {
	q := make(QEvent, 128)
	go peer.Transport.Receive(q)
	for event := range q {
		event.bind(peer)

		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			peer.PendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e := peer.PendingReplyEvents.Complete(features.InvocationID, event)
			if e == nil {
				peer.Transport.Send(newAcceptEvent(event))
			} else {
				log.Printf("[peer] %s (ID=%s event=%s)", e, peer.ID, event)
			}
		case NextEvent:
			features := event.Features()
			e := peer.PendingNextEvents.Complete(features.YieldID, event)
			if e == nil {
				peer.Transport.Send(newAcceptEvent(event))
			} else {
				log.Printf("[peer] %s (ID=%s event=%s)", e, peer.ID, event)
			}
		case PublishEvent:
			peer.producePublishEvent(event)
			peer.Transport.Send(newAcceptEvent(event))
		case CallEvent:
			peer.produceCallEvent(event)
			peer.Transport.Send(newAcceptEvent(event))
		default:
			log.Printf("[peer] InvalidEvent (ID=%s event=%s)", peer.ID, event)
		}
	}

	peer.closePublishEvents()
	peer.closeCallEvents()
}
