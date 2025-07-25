// Package multicloser provides a concurrency-safe mechanism for managing and closing
// multiple io.Closer instances.
//
// It is useful when you need to aggregate resource cleanup logic such as files, network
// connections, or custom closers, and ensure they are all closed reliably, even in
// the presence of errors.
//
// # Overview
//
// MultiCloser allows callers to register multiple io.Closer instances and later call
// Close() once to close all of them. It is safe for use by multiple goroutines.
// The zero state is not usable, use New() to construct an instance.
//
// # Reuse
//
// After Close is called, the MultiCloser can be reused to register new closers.
// Internally, the state is reset after each Close call.
//
// # Error Aggregation
//
// When Close is called, MultiCloser attempts to close all registered io.Closers.
// If any of them return errors, all errors are aggregated and returned as a single
// error using errors.Join, available in Go 1.20+.
//
// # Concurrency
//
// MultiCloser is safe for concurrent use. All operations — Register, Unregister, Close,
// and Len — are protected by a mutex. However, behavior is deterministic only in the
// absence of interleaved Register/Unregister/Close calls.
//
// For example, registering and unregistering the same closer concurrently may result
// in the closer being either closed or skipped, depending on the timing of Go's scheduler.
//
// # Example Usage
//
//	func doWork() error {
//	    mc := multicloser.New()
//
//	    f, err := os.Open("file.txt")
//	    if err != nil {
//	        return err
//	    }
//	    mc.Register(f)
//
//	    conn, err := net.Dial("tcp", "example.com:80")
//	    if err != nil {
//	        return err
//	    }
//	    mc.Register(conn)
//
//	    // perform operations...
//
//	    return mc.Close()
//	}
//
// # Errors
//
// If Close is called when no closers are registered, it returns multicloser.ErrNoCloserRegistered.
package multicloser
