package wamp

import (
	"errors"
	"log"
	"sync"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var (
	ConnectionLost = errors.New("ConnectionLost")
	TimedOut = errors.New("TimedOut")
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
	connectionLost               bool
	Alive                        chan struct{}
	writeMutex                   *sync.Mutex
	Transport                    Transport
	PendingAcceptEvents          *wampShared.PendingMap[AcceptEvent]
	PendingReplyEvents           *wampShared.PendingMap[ReplyEvent]
	PendingCancelEvents          *wampShared.PendingMap[CancelEvent]
	PendingNextEvents            *wampShared.PendingMap[NextEvent]
	producePublishEvent          wampShared.Producible[PublishEvent]
	ConsumeIncomingPublishEvents wampShared.Consumable[PublishEvent]
	closePublishEvents           wampShared.Closeable
	produceCallEvent             wampShared.Producible[CallEvent]
	ConsumeIncomingCallEvents    wampShared.Consumable[CallEvent]
	closeCallEvents              wampShared.Closeable
}

func (peer *Peer) safeSend(event Event) error {
	// prevent concurrent writes
	peer.writeMutex.Lock()
	e := peer.Transport.Write(event)
	peer.writeMutex.Unlock()
	return e
}

func (peer *Peer) acknowledge(source Event) error {
	acceptEvent := newAcceptEvent(source)
	e := peer.safeSend(acceptEvent)
	return e
}

func (peer *Peer) Send(event Event) error {
	if peer.connectionLost {
		return ConnectionLost
	}
	acceptEventPromise, _ := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	// TODO retry
	peer.safeSend(event)
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return TimedOut
}

func (peer *Peer) Close() error {
	e := peer.Transport.Close()
	return e
}

func newPeer(
	ID string,
	transport Transport,
) *Peer {
	consumePublishEvents, producePublishEvent, closePublishEvents := wampShared.NewStream[PublishEvent]()
	consumeCallEvents, produceCallEvent, closeCallEvents := wampShared.NewStream[CallEvent]()
	return &Peer{
		ID,
		false,
		make(chan struct{}),
		new(sync.Mutex),
		transport,
		wampShared.NewPendingMap[AcceptEvent](),
		wampShared.NewPendingMap[ReplyEvent](),
		wampShared.NewPendingMap[CancelEvent](),
		wampShared.NewPendingMap[NextEvent](),
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
		if e == nil {
			log.Printf("[peer] new event (ID=%s)", peer.ID)
		} else if e == ConnectionLost {
			log.Printf("[peer] connection lost (ID=%s)", peer.ID)
			break
		} else {
			log.Printf("[peer] transport error %s (ID=%s)", e, peer.ID)
			continue
		}

		event.setPeer(peer)

		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.PendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			e = errors.Join(
				peer.PendingReplyEvents.Complete(features.InvocationID, event),
				peer.acknowledge(event),
			)
		case PublishEvent:
			peer.producePublishEvent(event)
			e = peer.acknowledge(event)
		case CallEvent:
			peer.produceCallEvent(event)
			e = peer.acknowledge(event)
		case NextEvent:
			features := event.Features()
			e = errors.Join(
				peer.PendingNextEvents.Complete(features.YieldID, event),
				peer.acknowledge(event),
			)
		case CancelEvent:
			features := event.Features()
			e = errors.Join(
				peer.PendingCancelEvents.Complete(features.InvocationID, event),
				peer.acknowledge(event),
			)
		}

		if e == nil {
			log.Printf("[peer] event success (ID=%s)", peer.ID)
		} else {
			log.Printf("[peer] listening error %s (ID=%s)", e, peer.ID)
		}
	}

	peer.connectionLost = true
	peer.closePublishEvents()
	peer.closeCallEvents()
	close(peer.Alive)
}

func SpawnPeer(
	ID string,
	transport Transport,
) *Peer {
	peer := newPeer(ID, transport)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go listenEvents(wg, peer)
	wg.Wait()

	return peer
}
