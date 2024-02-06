package wampTransports

import (
	"sync"

	wamp "github.com/wamp3hub/wamp3go"
)

type ResumableTransport struct {
	open      bool
	reading   sync.Locker
	writing   sync.Locker
	transport wamp.Transport
}

func MakeResumable(transport wamp.Transport) *ResumableTransport {
	return &ResumableTransport{
		true,
		new(sync.Mutex),
		new(sync.Mutex),
		transport,
	}
}

// Pause IO operations
func (resumable *ResumableTransport) Pause() {
	if resumable.open {
		resumable.open = false
		resumable.writing.Lock()
		resumable.reading.Lock()
	}
}

// Resume IO operations
func (resumable *ResumableTransport) Resume() {
	if !resumable.open {
		resumable.open = true
		resumable.reading.Unlock()
		resumable.writing.Unlock()
	}
}

func (resumable *ResumableTransport) Close() error {
	return resumable.transport.Close()
}

func (resumable *ResumableTransport) Read() (wamp.Event, error) {
	// prevent concurrent reads
	resumable.reading.Lock()
	defer resumable.reading.Unlock()
	return resumable.transport.Read()
}

func (resumable *ResumableTransport) Write(event wamp.Event) error {
	resumable.writing.Lock()
	defer resumable.writing.Unlock()
	return resumable.transport.Write(event)
}
