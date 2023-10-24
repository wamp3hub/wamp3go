package wamp3go

import (
	"errors"
	"log"
	"sync"

	"github.com/wamp3hub/wamp3go/shared"
)

type Serializer interface {
	Code() string
	Encode(Event) ([]byte, error)
	Decode([]byte) (Event, error)
}

type Transport interface {
	Close() error
	Read() (Event, error)
	Write(Event) error
}

type Peer struct {
	ID                           string
	Alive                        chan struct{}
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
	e := peer.Transport.Write(event)
	peer.writeMutex.Unlock()
	return e
}

func (peer *Peer) acknowledge(source Event) error {
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

func newPeer(
	ID string,
	transport Transport,
) *Peer {
	consumePublishEvents, producePublishEvent, closePublishEvents := shared.NewStream[PublishEvent]()
	consumeCallEvents, produceCallEvent, closeCallEvents := shared.NewStream[CallEvent]()
	return &Peer{
		ID,
		make(chan struct{}),
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
	wg.Done()

	for {
		event, e := peer.Transport.Read()
		if e != nil {
			log.Printf("[peer] transport error %e (ID=%s)", e, peer.ID)
			break
		}

		event.setPeer(peer)

		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.PendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = peer.PendingReplyEvents.Complete(features.InvocationID, event)
			if e == nil {
				e = peer.acknowledge(event)
			}
		case NextEvent:
			features := event.Features()
			e = peer.PendingNextEvents.Complete(features.YieldID, event)
			if e == nil {
				e = peer.acknowledge(event)
			}
		case PublishEvent:
			peer.producePublishEvent(event)
			e = peer.acknowledge(event)
		case CallEvent:
			peer.produceCallEvent(event)
			e = peer.acknowledge(event)
		}

		if e != nil {
			log.Printf("[peer] listener error %e (ID=%s)", e, peer.ID)
		}
	}

	peer.closePublishEvents()
	peer.closeCallEvents()

	close(peer.Alive)
}

func SpawnPeer(ID string, transport Transport) *Peer {
	peer := newPeer(ID, transport)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go listenEvents(wg, peer)
	wg.Wait()

	return peer
}
