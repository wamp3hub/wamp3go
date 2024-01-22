package wamp

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

const (
	DEFAULT_TIMEOUT      = 60
	DEFAULT_RESEND_COUNT = 3
)

var (
	ErrorSerialization  = errors.New("serialization error")
	ErrorBadConnection  = errors.New("bad connection")
	ErrorConnectionLost = errors.New("connection lost")
	ErrorTimedOut       = errors.New("timed out")
)

type Serializer interface {
	Code() string
	Encode(Event) ([]byte, error)
	Decode([]byte) (Event, error)
}

type Transport interface {
	Connect() error
	Close() error
	Read() (Event, error)
	Write(Event) error
}

type Peer struct {
	ID                    string
	Alive                 chan struct{}
	writeMutex            *sync.Mutex
	readMutex             *sync.Mutex
	transport             Transport
	reconnectionStrategy  wampShared.RetryStrategy
	ReJoinEvents          *wampShared.ObservableObject[struct{}]
	pendingAcceptEvents   *wampShared.PendingMap[AcceptEvent]
	PendingReplyEvents    *wampShared.PendingMap[ReplyEvent]
	PendingCancelEvents   *wampShared.PendingMap[CancelEvent]
	PendingNextEvents     *wampShared.PendingMap[NextEvent]
	IncomingPublishEvents *wampShared.ObservableObject[PublishEvent]
	IncomingCallEvents    *wampShared.ObservableObject[CallEvent]
	logger                *slog.Logger
}

func newPeer(
	ID string,
	transport Transport,
	reconnectionStrategy wampShared.RetryStrategy,
	logger *slog.Logger,
) *Peer {
	return &Peer{
		ID,
		make(chan struct{}),
		new(sync.Mutex),
		new(sync.Mutex),
		transport,
		reconnectionStrategy,
		wampShared.NewObservable[struct{}](),
		wampShared.NewPendingMap[AcceptEvent](),
		wampShared.NewPendingMap[ReplyEvent](),
		wampShared.NewPendingMap[CancelEvent](),
		wampShared.NewPendingMap[NextEvent](),
		wampShared.NewObservable[PublishEvent](),
		wampShared.NewObservable[CallEvent](),
		logger.With(
			slog.Group(
				"peer",
				"ID", ID,
			),
		),
	}
}

// Pause io operations
func (peer *Peer) Pause() {
	peer.writeMutex.Lock()
	peer.readMutex.Lock()
	peer.logger.Debug("io pause")
}

// Resume io operations
func (peer *Peer) Resume() {
	peer.readMutex.Unlock()
	peer.writeMutex.Unlock()
	peer.logger.Debug("io resume")
}

// reconnects until the connection is established
func (peer *Peer) reconnect() error {
	if peer.reconnectionStrategy.AttemptNumber() == 0 {
		peer.Pause()
	}

	if peer.reconnectionStrategy.Done() {
		peer.logger.Error("reconnection attempts exceeded")
		return ErrorConnectionLost
	}

	sleepDuration := peer.reconnectionStrategy.Next()
	peer.logger.Debug("sleeping", "duration", sleepDuration)
	time.Sleep(sleepDuration)

	peer.logger.Warn("reconnecting...")
	e := peer.transport.Connect()
	if e == nil {
		peer.logger.Info("successfully reconnected")
		peer.reconnectionStrategy.Reset()
		peer.Resume()
		peer.ReJoinEvents.Next(struct{}{})
		return nil
	}

	peer.logger.Error("during reconnect", "error", e)
	return peer.reconnect()
}

// Closes the connection
func (peer *Peer) Close() error {
	peer.logger.Debug("trying to close...")
	e := peer.transport.Close()
	if e == nil {
		peer.logger.Debug("successfully closed")
	} else {
		peer.logger.Error("during close", "error", e)
	}
	return e
}

// prevent concurrent writes
// and locks writing if connection is lost
func (peer *Peer) safeSend(event Event) error {
	peer.writeMutex.Lock()
	e := peer.transport.Write(event)
	peer.writeMutex.Unlock()
	return e
}

// Sends an acknowledgement to the peer.
func (peer *Peer) acknowledge(source Event) bool {
	logData := slog.Group(
		"source",
		"ID", source.ID(),
		"Kind", source.Kind(),
	)
	acceptEvent := newAcceptEvent(source)
	for i := DEFAULT_RESEND_COUNT; i > -1; i-- {
		e := peer.safeSend(acceptEvent)
		if e == nil {
			peer.logger.Debug("acknowledgement successfully sent", logData)
			return true
		}
		peer.logger.Error("during send acknowledgement", "error", e, "i", i, logData)
	}
	return false
}

// Sends an event to the peer.
// Returns `true` if the event was successfully delivered.
// `retryCount` is the number of times the event will be resent.
func (peer *Peer) Send(event Event, retryCount int) bool {
	if retryCount < 0 {
		return false
	}

	logData := slog.Group(
		"event",
		"ID", event.ID(),
		"Kind", event.Kind(),
	)
	peer.logger.Debug("trying to send", logData)

	// creates a timeless promise, because the timeout is manually
	acceptEventPromise, cancelAcceptEventPromise := peer.pendingAcceptEvents.New(event.ID(), 0)

	e := peer.safeSend(event)
	if e == nil {
		peer.logger.Debug("event successfully sent", logData)
		select {
		case <-acceptEventPromise:
			peer.logger.Debug("event successfully delivered", logData)
			return true
		case <-time.After(DEFAULT_TIMEOUT * time.Second):
			e = errors.New("event not delivered (TimedOut)")
		}
	}

	peer.logger.Error("during send", "error", e, logData)
	cancelAcceptEventPromise()
	return peer.Send(event, retryCount-1)
}

// locks reading if the connection is lost
func (peer *Peer) safeRead() (Event, error) {
	peer.readMutex.Lock()
	event, e := peer.transport.Read()
	peer.readMutex.Unlock()
	return event, e
}

func (peer *Peer) readIncomingEvents(wg *sync.WaitGroup) {
	peer.logger.Debug("reading begin")
	wg.Done()

	for {
		event, e := peer.safeRead()
		if e != nil {
			if errors.Is(e, ErrorBadConnection) {
				peer.logger.Warn("bad connection")
				e = peer.reconnect()
			}

			if errors.Is(e, ErrorConnectionLost) {
				peer.logger.Warn("connection lost")
				break
			}

			peer.logger.Warn("during read", "error", e)
			// TODO count errors
			continue
		}

		logData := slog.Group(
			"event",
			"ID", event.ID(),
			"Kind", event.Kind(),
		)
		peer.logger.Debug("new", logData)

		event.setPeer(peer)

		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.pendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			peer.acknowledge(event)
			e = peer.PendingReplyEvents.Complete(features.InvocationID, event)
		case PublishEvent:
			peer.IncomingPublishEvents.Next(event)
			peer.acknowledge(event)
		case CallEvent:
			peer.IncomingCallEvents.Next(event)
			peer.acknowledge(event)
		case NextEvent:
			features := event.Features()
			peer.acknowledge(event)
			e = peer.PendingNextEvents.Complete(features.YieldID, event)
		case CancelEvent:
			features := event.Features()
			peer.acknowledge(event)
			e = peer.PendingCancelEvents.Complete(features.InvocationID, event)
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
	close(peer.Alive)

	peer.logger.Debug("reading end")
}

func SpawnPeer(
	ID string,
	transport Transport,
	reconnectionStrategy wampShared.RetryStrategy,
	logger *slog.Logger,
) *Peer {
	peer := newPeer(ID, transport, reconnectionStrategy, logger)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go peer.readIncomingEvents(wg)
	wg.Wait()

	return peer
}
