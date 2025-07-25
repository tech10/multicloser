package multicloser

import (
	"errors"
	"io"
	"sync"
)

// Error indicating if the multicloser is already closed.
var ErrNoCloserRegistered = errors.New("no io.Closer(s) registered")

// MultiCloser manages multiple io.Closers safely.
//
// This is safe for concurrent use by multiple goroutines.
// Once closed, this can be reused by registering more io.Closers.
type MultiCloser struct {
	mu      sync.Mutex
	closers map[io.Closer]struct{}
}

// New creates and returns a new MultiCloser.
func New() *MultiCloser {
	return &MultiCloser{
		closers: make(map[io.Closer]struct{}),
	}
}

// Register adds an io.Closer to the MultiCloser.
func (m *MultiCloser) Register(c io.Closer) {
	if c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closers[c] = struct{}{}
}

// Unregister removes an io.Closer from the MultiCloser.
func (m *MultiCloser) Unregister(c io.Closer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.closers, c)
}

// Close implements the io.Closer interface. It closes all registered closers.
// If no closers are registered, it returns an error.
func (m *MultiCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.closers) == 0 {
		return ErrNoCloserRegistered
	}

	var errs []error
	for c := range m.closers {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	// reset for reuse
	m.closers = make(map[io.Closer]struct{})

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Returns the number of io.Closer(s) registered.
func (m *MultiCloser) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.closers)
}
