package output

import (
	"context"
	"io"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type BestEffortWriter struct {
	fd int
}

func NewBestEffortWriter(fd int) *BestEffortWriter {
	return &BestEffortWriter{
		fd: fd,
	}
}

func (w *BestEffortWriter) writeOnce(p []byte) (int, error) {
	n, err := unix.Write(w.fd, p)
	if n < 0 {
		n = 0
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			err = nil
		}
	}
	return n, err
}

func (w *BestEffortWriter) Write(b []byte) (n int, err error) {
	for n < len(b) {
		var n2 int
		n2, err = w.writeOnce(b[n:])
		n += n2
		if err != nil || n2 <= 0 {
			break
		}
	}

	return
}

type GoWriter struct {
	sync.Mutex
	w      io.Writer
	inChan chan []byte
	wg     sync.WaitGroup
	err    error
}

func NewGoWriter(ctx context.Context, w io.Writer) *GoWriter {
	gw := &GoWriter{
		w:      w,
		inChan: make(chan []byte),
	}

	go func() {
		iterNoErr := func() {
			select {
			case <-ctx.Done():
				gw.setErr(ctx.Err())

			case b, ok := <-gw.inChan:
				if ok {
					gw.writeNow(b)
				}
			}
		}

		for gw.Err() == nil {
			iterNoErr()
		}
		for b := range gw.inChan {
			gw.writeNow(b)
		}
	}()

	return gw
}

func (gw *GoWriter) Write(b []byte) (int, error) {
	err := gw.acquire()
	if err != nil {
		return 0, err
	}

	gw.inChan <- b
	return len(b), nil
}

func (gw *GoWriter) Err() error {
	gw.Lock()
	defer gw.Unlock()

	return gw.err
}

func (gw *GoWriter) acquire() error {
	gw.Lock()
	defer gw.Unlock()

	if gw.err != nil {
		return gw.err
	}

	gw.wg.Add(1)
	return nil
}

func (gw *GoWriter) release() {
	gw.wg.Done()
}

func (gw *GoWriter) setErr(err error) error {
	gw.Lock()
	defer gw.Unlock()

	if gw.err != nil {
		// Already stopped.
		return gw.err
	}

	gw.err = err

	go func() {
		gw.wg.Wait()
		close(gw.inChan)
	}()

	return nil
}

func (gw *GoWriter) writeNow(b []byte) {
	defer gw.release()
	_, err := gw.w.Write(b)
	if err != nil {
		gw.setErr(err)
	}
}

type SyncWriter struct {
	gws []*GoWriter
}

func NewSyncWriter(ctx context.Context, ws []io.Writer) *SyncWriter {
	var gws []*GoWriter
	for _, w := range ws {
		gws = append(gws, NewGoWriter(ctx, w))
	}

	return &SyncWriter{
		gws: gws,
	}
}

// Write writes the given bytes to each of the sync writer's constituent
// writers in parallel, then waits for all the writes to complete.
func (sw *SyncWriter) Write(b []byte) (int, error) {
	for _, gw := range sw.gws {
		n, err := gw.Write(b)
		if err != nil {
			return n, err
		}
	}

	sw.wait()
	return len(b), nil
}

func (sw *SyncWriter) wait() {
	for _, gw := range sw.gws {
		gw.wg.Wait()
	}
}
