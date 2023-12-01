package wamp

import (
	"errors"
	"log"
	"sync"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var (
	ConnectionLostError = errors.New("ConnectionLost")
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
	ID                    string
	connectionLost        bool
	Alive                 chan struct{}
	writeMutex            *sync.Mutex
	Transport             Transport
	PendingAcceptEvents   *wampShared.PendingMap[AcceptEvent]
	PendingReplyEvents    *wampShared.PendingMap[ReplyEvent]
	PendingCancelEvents   *wampShared.PendingMap[CancelEvent]
	PendingNextEvents     *wampShared.PendingMap[NextEvent]
	IncomingPublishEvents *wampShared.ObservableObject[PublishEvent]
	IncomingCallEvents    *wampShared.ObservableObject[CallEvent]
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
		return ConnectionLostError
	}
	acceptEventPromise, _ := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT)
	// TODO retry
	e := peer.safeSend(event)
	if e == nil {
		log.Printf("[peer] event sent (ID=%s event.Kind=%d)", peer.ID, event.Kind())
	}
	_, done := <-acceptEventPromise
	if done {
		return nil
	}
	return TimedOutError
}

func (peer *Peer) Close() error {
	e := peer.Transport.Close()
	return e
}

func newPeer(
	ID string,
	transport Transport,
) *Peer {
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
		wampShared.NewObservable[PublishEvent](),
		wampShared.NewObservable[CallEvent](),
	}
}

func listenEvents(wg *sync.WaitGroup, peer *Peer) {
	wg.Done()

	for {
		event, e := peer.Transport.Read()
		if e == nil {
			log.Printf("[peer] new event (ID=%s event.Kind=%d)", peer.ID, event.Kind())
		} else if e == ConnectionLostError {
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
			peer.IncomingPublishEvents.Next(event)
			e = peer.acknowledge(event)
		case CallEvent:
			peer.IncomingCallEvents.Next(event)
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
			log.Printf("[peer] event success (ID=%s Kind=%d)", peer.ID, event.Kind())
		} else {
			log.Printf("[peer] listening error %s (ID=%s Kind=%d)", e, peer.ID, event.Kind())
		}
	}

	peer.IncomingPublishEvents.Complete()
	peer.IncomingCallEvents.Complete()
	peer.connectionLost = true
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
