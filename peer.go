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

func (peer *Peer) send(event Event) error {
	// avoid concurrent write
	peer.writeMutex.Lock()
	e := peer.Transport.Send(event)
	peer.writeMutex.Unlock()
	return e
}

func (peer *Peer) sendAccept(source Event) error {
	acceptEvent := newAcceptEvent(source)
	e := peer.send(acceptEvent)
	return e
}

func (peer *Peer) Say(event Event) error {
	acceptEventPromise, _ := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	peer.send(event)
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return errors.New("TimedOut")
}

func (peer *Peer) Close() error {
	e := peer.Transport.Close()
	return e
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

func listenEvents(wg *sync.WaitGroup, peer *Peer) {
	q := make(QEvent, 128)
	go peer.Transport.Receive(q)
	// transport must send first empty event
	<-q
	wg.Done()
	for event := range q {
		event.setPeer(peer)

		e := error(nil)
		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.PendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = peer.PendingReplyEvents.Complete(features.InvocationID, event)
			if e == nil {
				e = peer.sendAccept(event)
			}
		case NextEvent:
			features := event.Features()
			e = peer.PendingNextEvents.Complete(features.YieldID, event)
			if e == nil {
				e = peer.sendAccept(event)
			}
		case PublishEvent:
			peer.producePublishEvent(event)
			e = peer.sendAccept(event)
		case CallEvent:
			peer.produceCallEvent(event)
			e = peer.sendAccept(event)
		}

		if e == nil {
			log.Printf("[peer] success (ID=%s event.ID=%s)", peer.ID, event.ID())
		} else {
			log.Printf("[peer] error %e (ID=%s)", e, peer.ID)
		}
	}

	peer.closePublishEvents()
	peer.closeCallEvents()
}

func SpawnPeer(ID string, transport Transport) *Peer {
	peer := NewPeer(ID, transport)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go listenEvents(wg, peer)
	wg.Wait()

	return peer
}
