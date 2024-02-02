package wamp

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	wampShared "github.com/wamp3hub/wamp3go/shared"
)

const (
	default_send_timeout = 60
	default_resend_count = 3
)

var (
	ErrorSerialization      = errors.New("serialization error")
	ErrorConnectionRestored = errors.New("connection was restored")
	ErrorConnectionClosed   = errors.New("connection is closed")
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
	transport             Transport
	RejoinEvents          *wampShared.Observable[struct{}]
	pendingAcceptEvents   *wampShared.PendingMap[AcceptEvent]
	PendingReplyEvents    *wampShared.PendingMap[ReplyEvent]
	PendingCancelEvents   *wampShared.PendingMap[CancelEvent]
	PendingNextEvents     *wampShared.PendingMap[NextEvent]
	IncomingPublishEvents *wampShared.Observable[PublishEvent]
	IncomingCallEvents    *wampShared.Observable[CallEvent]
	logger                *slog.Logger
}

func newPeer(
	ID string,
	transport Transport,
	logger *slog.Logger,
) *Peer {
	return &Peer{
		ID,
		transport,
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

// Sends an acknowledgement to the peer.
func (peer *Peer) acknowledge(source Event) bool {
	logData := slog.Group(
		"source",
		"ID", source.ID(),
		"Kind", source.Kind(),
	)
	acceptEvent := newAcceptEvent(source)
	for i := default_resend_count; i > -1; i-- {
		e := peer.transport.Write(acceptEvent)
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

	// creates a promise that does not expire
	acceptEventPromise, cancelAcceptEventPromise := peer.pendingAcceptEvents.New(event.ID(), 0)

	e := peer.transport.Write(event)
	if e == nil {
		peer.logger.Debug("event successfully sent", logData)

		select {
		case <-acceptEventPromise:
			peer.logger.Debug("event successfully delivered", logData)
			return true
		case <-time.After(default_send_timeout * time.Second):
			e = errors.New("event not delivered (TimedOut)")
		}
	}

	peer.logger.Error("during send", "error", e, logData)
	cancelAcceptEventPromise()
	return peer.Send(event, retryCount-1)
}

func (peer *Peer) readIncomingEvents(wg *sync.WaitGroup) {
	peer.logger.Debug("reading incoming events begin")
	wg.Done()

	for {
		event, e := peer.transport.Read()
		if e != nil {
			if errors.Is(e, ErrorConnectionClosed) {
				peer.logger.Warn("connection lost")
				break
			}

			if errors.Is(e, ErrorConnectionRestored) {
				peer.logger.Warn("bad connection")
				peer.RejoinEvents.Next(struct{}{})
			}

			peer.logger.Warn("during read event", "error", e)
			// TODO count errors
			// TODO rate limit if error count exceeded
			continue
		}

		event.setRouter(peer)

		logData := slog.Group(
			"event",
			"ID", event.ID(),
			"Kind", event.Kind(),
		)
		peer.logger.Debug("new event", logData)

		// TODO exclude duplicates

		switch event := event.(type) {
		case AcceptEvent:
			features := event.Features()
			e = peer.pendingAcceptEvents.Complete(features.SourceID, event)
		case ReplyEvent:
			features := event.Features()
			peer.acknowledge(event)
			e = peer.PendingReplyEvents.Complete(features.InvocationID, event)
		case PublishEvent:
			go peer.IncomingPublishEvents.Next(event)
			peer.acknowledge(event)
		case CallEvent:
			go peer.IncomingCallEvents.Next(event)
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
			peer.logger.Debug("read event success", logData)
		} else {
			peer.logger.Error("during read event", "error", e, logData)
		}
	}

	peer.IncomingPublishEvents.Complete()
	peer.IncomingCallEvents.Complete()
	peer.RejoinEvents.Complete()

	peer.logger.Debug("reading incoming events end")
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
