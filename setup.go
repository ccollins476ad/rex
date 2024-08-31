package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/badvassal/rex/dest"
)

type Env struct {
	ReadBufSize int
	Writers     []io.Writer
}

func parseArgs() (*Env, error) {
	readBufSize := flag.Int("b", 64*1024, "read buffer size")
	flag.Parse()

	// All remaining arguments specify destinations. Parse each and append
	// their corresponding writers to ws.
	var ws []io.Writer
	for _, arg := range flag.Args() {
		fail := func(err error) (*Env, error) {
			return nil, fmt.Errorf(`failed to process argument "%s": %w`, arg, err)
		}

		d, err := dest.Parse(arg)
		if err != nil {
			return fail(err)
		}

		w, err := d.Open()
		if err != nil {
			return fail(err)
		}

		ws = append(ws, w)
	}

	if len(ws) == 0 {
		return nil, fmt.Errorf("at least one output required")
	}

	return &Env{
		ReadBufSize: *readBufSize,
		Writers:     ws,
	}, nil
}
