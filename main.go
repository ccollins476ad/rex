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

// fatal prints the given error to stderr and terminates with an appropriate
// status.
func fatal(err error, printUsage bool) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	if printUsage {
		fmt.Fprintln(os.Stderr)
		flag.CommandLine.Usage()
	}

	exitStatus := 1 // Exit with 1 by default.

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

	sw := output.NewSyncWriter(ctx, env.Writers)

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
