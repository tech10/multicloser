# multicloser

[![Go Reference](https://pkg.go.dev/badge/github.com/tech10/multicloser.svg)](https://pkg.go.dev/github.com/tech10/multicloser)
[![Go Report Card](https://goreportcard.com/badge/github.com/tech10/multicloser)](https://goreportcard.com/report/github.com/tech10/multicloser)

`multicloser` is a tiny, concurrency-safe Go library that manages multiple `io.Closer` instances. It provides a way to register multiple closers and close them all at once safely and reliably across multiple goroutines.

## Features

- Does not import other dependencies.
- Small, fast, and created for only its specific purpose, no extra addons.
- Register multiple `io.Closer`s
- Call `Close()` once to close all registered resources
- Can be reused after calling `Close()`
- Aggregates multiple close errors using `errors.Join` (Go 1.20+)
- Fully concurrency-safe

## Use case

Among other things, this library could be used to close multiple network connections, files, and custom `io.Closer`s at the same time after capturing a signal, such as with the use of `signal.Notify`.

## Installation

```bash
go get github.com/tech10/multicloser
```

## Usage

```go
package main

import (
	"net"
	"os"

	"github.com/tech10/multicloser"
)

func main() {
	mc := multicloser.New()

	f, err := os.Open("file.txt")
	if err != nil {
		panic(err) // do not actually panic, a good program never panics on errors of this type
	}
	mc.Register(f)

	conn, err := net.Dial("tcp", "example.com:80")
	if err != nil {
		panic(err)
	}
	mc.Register(conn)

	// Do something with the resources here.

	if err := mc.Close(); err != nil {
		// Handle close errors (may be multiple as joined by errors.Join internally)
		panic(err)
	}
}
```

## Documentation

View full documentation at [pkg.go.dev/github.com/tech10/multicloser](https://pkg.go.dev/github.com/tech10/multicloser)
