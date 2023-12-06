package wamp

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

const DEFAULT_TIMEOUT = 60

var (
	ErrorConnectionLost = errors.New("ConnectionLost")
	ErrorSerialization  = errors.New("SerializationError")
	ErrorTimedOut       = errors.New("TimedOut")
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
	logger                *slog.Logger
}

func (peer *Peer) safeSend(event Event) error {
	// prevent concurrent writes
	peer.writeMutex.Lock()
	e := peer.Transport.Write(event)
	peer.writeMutex.Unlock()

	// TODO retry

	return e
}

func (peer *Peer) acknowledge(source Event) error {
	logData := slog.Group("source", "ID", source.ID(), "Kind", source.Kind())
	acceptEvent := newAcceptEvent(source)
	e := peer.safeSend(acceptEvent)
	if e == nil {
		peer.logger.Debug("acknowledge successfully sent", logData)
	} else {
		peer.logger.Error("during acknowledgement", "error", e, logData)
	}
	return e
}

func (peer *Peer) Send(event Event) error {
	if peer.connectionLost {
		peer.logger.Error("connection already closed")
		return ErrorConnectionLost
	}

	acceptEventPromise, _ := peer.PendingAcceptEvents.New(event.ID(), DEFAULT_TIMEOUT*time.Second)

	logData := slog.Group("event", "ID", event.ID(), "Kind", event.Kind())
	peer.logger.Debug("sending", logData)

	e := peer.safeSend(event)
	if e == nil {
		peer.logger.Debug("event successfully sent", "event.Kind", event.Kind())
	} else {
		peer.logger.Error("during send", "error", e, logData)
	}

	_, done := <-acceptEventPromise
	if done {
		return nil
	}

	peer.logger.Error("failed to deliver event (Timedout)", logData)
	return ErrorTimedOut
}

func (peer *Peer) readIncomingEvents(wg *sync.WaitGroup) {
	peer.logger.Debug("reading begin")
	wg.Done()

	for {
		event, e := peer.Transport.Read()
		if e != nil {
			if errors.Is(e, ErrorConnectionLost) {
				peer.logger.Warn("connection lost")
				break
			}

			peer.logger.Warn("during read", "error", e)
			// TODO count errors
			continue
		}

		logData := slog.Group("event", "ID", event.ID(), "Kind", event.Kind())
		peer.logger.Debug("new", logData)

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
		default:
			e = errors.New("unexpected event type (ignoring)")
		}

		if e == nil {
			peer.logger.Debug("read success", logData)
		} else {
			peer.logger.Error("during read", "error", e, logData)
		}
	}

	peer.IncomingPublishEvents.Complete()
	peer.IncomingCallEvents.Complete()
	peer.connectionLost = true
	close(peer.Alive)

	peer.logger.Debug("reading end")
}

func (peer *Peer) Close() error {
	peer.logger.Debug("closing")
	e := peer.Transport.Close()
	if e == nil {
		peer.logger.Debug("successfully closed")
	} else {
		peer.logger.Error("during close", "error", e)
	}
	return e
}

func newPeer(
	ID string,
	transport Transport,
	logger *slog.Logger,
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
		logger.With(
			slog.Group("peer", "ID", ID),
		),
	}
}

func SpawnPeer(
	ID string,
	transport Transport,
	logger *slog.Logger,
) *Peer {
	peer := newPeer(ID, transport, logger)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go peer.readIncomingEvents(wg)
	wg.Wait()

	return peer
}
