package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/badvassal/rex/output"
)

var readBufSize int

// fatal optionally prints an error to stderr, optionally prints the rex usage
// text, and terminates with an appropriate status. It prints an error if
// err!=nil. It prints usage text if printUsage==true.
func fatal(err error, printUsage bool) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	if printUsage {
		fmt.Fprintln(os.Stderr)
		flag.CommandLine.Usage()
	}

	exitStatus := 1 // Exit with 1 by default.

	// If the error resulted from a posix call, terminate with the error's
	// status code.
	var errno *syscall.Errno
	if errors.As(err, &errno) {
		exitStatus = int(*errno)
	}

	os.Exit(exitStatus)
}

func main() {
	env, err := parseArgs()
	if err != nil {
		fatal(err, true)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build a synchronized writer that emits output to all destinations
	// specified on the command line.
	sw := output.NewSyncWriter(ctx, env.Writers)

	// Continuously read from stdin, then use the synchronized writer to write
	// the data to all destinations in parallel.
	buf := make([]byte, env.ReadBufSize)
	for {
		n, readErr := os.Stdin.Read(buf)

		if n > 0 {
			_, err := sw.Write(buf[:n])
			if err != nil {
				fatal(err, false)
			}
		}

		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				fatal(fmt.Errorf("read: %w", readErr), false)
			}
			break
		}
	}
}
