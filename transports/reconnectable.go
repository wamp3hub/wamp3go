package wampTransports

import (
	"errors"
	"log/slog"
	"time"

	wamp "github.com/wamp3hub/wamp3go"
	wampShared "github.com/wamp3hub/wamp3go/shared"
)

var ErrorBadConnection = errors.New("bad connection")

// Wraps a transport and reconnects it when read returns `ErrorBadConnection`
type reconnectableTransport struct {
	resumable *ResumableTransport
	strategy  wampShared.RetryStrategy
	connect   func() (wamp.Transport, error)
	logger    *slog.Logger
}

func MakeReconnectable(
	connectFactory func() (wamp.Transport, error),
	strategy wampShared.RetryStrategy,
	logger *slog.Logger,
) (*reconnectableTransport, error) {
	instance := reconnectableTransport{
		MakeResumable(nil),
		strategy,
		connectFactory,
		logger.With(
			"name", "reconnectable",
		),
	}

	e := instance.reconnect()
	if errors.Is(e, wamp.ErrorConnectionRestored) {
		e = nil
	}

	return &instance, e
}

func (reconnectable *reconnectableTransport) Close() error {
	return reconnectable.resumable.Close()
}

func (reconnectable *reconnectableTransport) hotSwap(
	newTransport wamp.Transport,
) {
	if reconnectable.resumable.transport != nil {
		// close previous transport
		e := reconnectable.Close()
		if e == nil {
			reconnectable.logger.Debug("broken transport successfully closed")
		} else {
			reconnectable.logger.Warn("during close broken transport", "error", e)
		}
	}
	reconnectable.resumable.transport = newTransport
}

func (reconnectable *reconnectableTransport) reconnect() error {
	if reconnectable.strategy.AttemptNumber() == 0 {
		reconnectable.resumable.Pause()
	}

	if reconnectable.strategy.Done() {
		reconnectable.logger.Error("reconnection attempts exceeded")
		return wamp.ErrorConnectionClosed
	}

	sleepDuration := reconnectable.strategy.Next()
	if sleepDuration > 0 {
		reconnectable.logger.Debug("sleeping...", "duration", sleepDuration)
		time.Sleep(sleepDuration)
	}

	reconnectable.logger.Info("connecting...")
	newTransport, e := reconnectable.connect()
	if e != nil {
		reconnectable.logger.Error("during connect", "error", e)
		return reconnectable.reconnect()
	}

	reconnectable.logger.Info("successfully connected")

	reconnectable.strategy.Reset()

	reconnectable.hotSwap(newTransport)

	reconnectable.resumable.Resume()

	return wamp.ErrorConnectionRestored
}

func (reconnectable *reconnectableTransport) Read() (wamp.Event, error) {
	// prevent concurrent reads
	event, e := reconnectable.resumable.Read()
	if errors.Is(e, ErrorBadConnection) {
		e = reconnectable.reconnect()
	}
	return event, e
}

func (reconnectable *reconnectableTransport) Write(event wamp.Event) error {
	return reconnectable.resumable.Write(event)
}
