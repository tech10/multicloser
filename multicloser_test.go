package multicloser_test

import (
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/tech10/multicloser"
)

type mockCloser struct {
	closed bool
	err    error
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.err
}

func TestRegisterAndClose(t *testing.T) {
	mc := multicloser.New()

	c1 := &mockCloser{}
	c2 := &mockCloser{}

	mc.Register(c1)
	mc.Register(c2)

	if got := mc.Len(); got != 2 {
		t.Errorf("expected 2 closers registered, got %d", got)
	}

	err := mc.Close()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !c1.closed || !c2.closed {
		t.Errorf("expected all closers to be closed")
	}

	if got := mc.Len(); got != 0 {
		t.Errorf("expected 0 closers after close, got %d", got)
	}
}

func TestCloseWithError(t *testing.T) {
	mc := multicloser.New()

	err1 := errors.New("fail1")
	err2 := errors.New("fail2")

	c1 := &mockCloser{err: err1}
	c2 := &mockCloser{err: err2}
	c3 := &mockCloser{}

	mc.Register(c1)
	mc.Register(c2)
	mc.Register(c3)

	err := mc.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check joined error contains both error strings
	if !errors.Is(err, err1) || !errors.Is(err, err2) {
		t.Errorf("expected joined error to contain err1 and err2, got: %v", err)
	}
}

func TestRegisterNil(t *testing.T) {
	mc := multicloser.New()
	mc.Register(nil)

	if got := mc.Len(); got != 0 {
		t.Errorf("expected 0 closers, got %d", got)
	}
}

func TestUnregister(t *testing.T) {
	mc := multicloser.New()

	c1 := &mockCloser{}
	c2 := &mockCloser{}

	mc.Register(c1)
	mc.Register(c2)
	mc.Unregister(c1)

	if got := mc.Len(); got != 1 {
		t.Errorf("expected 1 closer after unregister, got %d", got)
	}

	_ = mc.Close()

	if c1.closed {
		t.Errorf("expected unregistered closer not to be closed")
	}
	if !c2.closed {
		t.Errorf("expected registered closer to be closed")
	}
}

func TestCloseEmpty(t *testing.T) {
	mc := multicloser.New()
	err := mc.Close()
	if !errors.Is(err, multicloser.ErrNoCloserRegistered) {
		t.Errorf("expected ErrNoClosers, got: %v", err)
	}
}

func TestReuseAfterClose(t *testing.T) {
	mc := multicloser.New()

	c1 := &mockCloser{}
	mc.Register(c1)
	_ = mc.Close()

	// Reuse
	c2 := &mockCloser{}
	mc.Register(c2)

	if got := mc.Len(); got != 1 {
		t.Errorf("expected 1 closer after reuse, got %d", got)
	}

	_ = mc.Close()

	if !c2.closed {
		t.Errorf("expected second closer to be closed")
	}
}

func TestConcurrentRegisterAndClose(t *testing.T) {
	const numClosers = 100
	const numWorkers = 10

	mc := multicloser.New()

	var wg sync.WaitGroup

	// Register closers concurrently
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numClosers/numWorkers; j++ {
				mc.Register(&mockCloser{})
			}
		}()
	}

	wg.Wait()

	if got := mc.Len(); got != numClosers {
		t.Errorf("expected %d closers registered, got %d", numClosers, got)
	}

	errs := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mc.Close()
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	// Only one Close() should return nil, the others may return ErrNoClosers
	success := 0
	for err := range errs {
		if err == nil {
			success++
		} else if !errors.Is(err, multicloser.ErrNoCloserRegistered) {
			t.Errorf("unexpected error: %v", err)
		}
	}

	if success != 1 {
		t.Errorf("expected only one successful Close(), got %d", success)
	}
}

func TestConcurrentUnregister(t *testing.T) {
	const numClosers = 100

	mc := multicloser.New()

	// Create and register 100 closers
	closers := make([]*mockCloser, numClosers)
	for i := range closers {
		closers[i] = &mockCloser{}
		mc.Register(closers[i])
	}

	if got := mc.Len(); got != numClosers {
		t.Fatalf("expected %d closers registered, got %d", numClosers, got)
	}

	var wg sync.WaitGroup

	// Concurrently unregister half of them
	for i := 0; i < numClosers; i += 2 {
		wg.Add(1)
		go func(c io.Closer) {
			defer wg.Done()
			mc.Unregister(c)
		}(closers[i])
	}

	wg.Wait()

	// Call Close to trigger actual closing
	err := mc.Close()
	if err != nil && !errors.Is(err, multicloser.ErrNoCloserRegistered) {
		t.Fatalf("unexpected error on close: %v", err)
	}

	// Check only the remaining half were closed
	for i := 0; i < numClosers; i++ {
		expectedClosed := i%2 != 0 // odd-indexed closers should still be registered
		if closers[i].closed != expectedClosed {
			t.Errorf("closer[%d] closed = %v, want %v", i, closers[i].closed, expectedClosed)
		}
	}
}
