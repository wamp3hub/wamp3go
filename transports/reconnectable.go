package wampTransports

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	wamp "github.com/wamp3hub/wamp3go"
	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var ErrorBadConnection = errors.New("bad connection")

// Wraps a transport and reconnects it when read returns `ErrorBadConnection`
type reconnectableTransport struct {
	open     bool
	reading  sync.Locker
	writing  sync.Locker
	base     wamp.Transport
	strategy wampShared.RetryStrategy
	connect  func() (wamp.Transport, error)
	logger   *slog.Logger
}

func NewReconnectableTransport(
	base wamp.Transport,
	strategy wampShared.RetryStrategy,
	connect func() (wamp.Transport, error),
	logger *slog.Logger,
) *reconnectableTransport {
	return &reconnectableTransport{
		true,
		new(sync.Mutex),
		new(sync.Mutex),
		base,
		strategy,
		connect,
		logger.With(
			"name", "reconnectable",
		),
	}
}

// Pause IO operations
func (reconnectable *reconnectableTransport) Pause() {
	if reconnectable.open {
		reconnectable.open = false
		reconnectable.writing.Lock()
		reconnectable.reading.Lock()
	}
}

// Resume IO operations
func (reconnectable *reconnectableTransport) Resume() {
	if !reconnectable.open {
		reconnectable.open = true
		reconnectable.reading.Unlock()
		reconnectable.writing.Unlock()
	}
}

func (reconnectable *reconnectableTransport) Close() error {
	return reconnectable.base.Close()
}

// Swap the underlying transport
func (reconnectable *reconnectableTransport) hotSwap(newTransport wamp.Transport) {
	reconnectable.Pause()
	// close previous transport
	reconnectable.Close()
	reconnectable.base = newTransport
	reconnectable.Resume()
}

func (reconnectable *reconnectableTransport) reconnect() error {
	if reconnectable.strategy.AttemptNumber() == 0 {
		reconnectable.Pause()
	}

	if reconnectable.strategy.Done() {
		reconnectable.logger.Error("reconnection attempts exceeded")
		return wamp.ErrorConnectionClosed
	}

	sleepDuration := reconnectable.strategy.Next()
	reconnectable.logger.Debug("sleeping...", "duration", sleepDuration)
	time.Sleep(sleepDuration)

	reconnectable.logger.Warn("reconnecting...")
	newTransport, e := reconnectable.connect()
	if e != nil {
		reconnectable.logger.Error("during reconnect", "error", e)
		return reconnectable.reconnect()
	}

	reconnectable.logger.Info("successfully reconnected")
	reconnectable.strategy.Reset()
	reconnectable.hotSwap(newTransport)
	return wamp.ErrorConnectionRestored
}

func (reconnectable *reconnectableTransport) Read() (wamp.Event, error) {
	// prevent concurrent reads
	reconnectable.reading.Lock()
	event, e := reconnectable.base.Read()
	reconnectable.reading.Unlock()
	if errors.Is(e, ErrorBadConnection) {
		e = reconnectable.reconnect()
	}
	return event, e
}

func (reconnectable *reconnectableTransport) Write(event wamp.Event) error {
	reconnectable.writing.Lock()
	defer reconnectable.writing.Unlock()
	return reconnectable.base.Write(event)
}
