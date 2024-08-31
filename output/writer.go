package output

import (
	"context"
	"fmt"
	"io"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// BestEffortWriter implements io.Writer. It writes to a unix file descriptor,
// treating EAGAIN and EWOULDBLOCK results as successes. That is, if the
// destination file reaches capacity before the full write completes, this
// function discards the unwritten data and reports success.
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
	if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
		// Zero bytes written due to full buffer. Treat these errors as
		// successes.
		return len(p), nil
	}
	if err != nil {
		// Fatal error.
		return 0, err
	}

	// Some bytes written. Not a failure.
	return n, nil
}

func (w *BestEffortWriter) Write(b []byte) (int, error) {
	var n int

	for n < len(b) {
		n2, err := w.writeOnce(b[n:])
		n += n2

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

// AsyncWriter implements io.Writer. It performs nonblocking writes in a
// dedicated goroutine. It is not possible to determine the results of a write
// operation.
type AsyncWriter struct {
	sync.Mutex
	w      io.Writer
	inChan chan []byte
	wg     sync.WaitGroup
	err    error
}

func NewAsyncWriter(ctx context.Context, w io.Writer) *AsyncWriter {
	aw := &AsyncWriter{
		w:      w,
		inChan: make(chan []byte),
	}

	go func() {
		// Consume from the input channel and write the contents until we
		// encounter an error.
		for aw.Err() == nil {
			select {
			case <-ctx.Done():
				aw.stop(ctx.Err())

			case b := <-aw.inChan:
				err := aw.writeNow(b)
				if err != nil {
					aw.stop(err)
				}
			}
		}

		// The writer is done (in the stopped state). Complete all pending
		// writes, then return.
		for b := range aw.inChan {
			aw.writeNow(b)
		}
	}()

	return aw
}

// Write schedules a write operation to run in the writer's goroutine. It
// returns an error if the writer is not accepting new writes (i.e., in the
// stopped state), otherwise it indicates success. The success return values
// can be misleading, since the scheduled write has not actually completed yet.
func (aw *AsyncWriter) Write(b []byte) (int, error) {
	err := aw.acquire()
	if err != nil {
		return 0, err
	}

	aw.inChan <- b
	return len(b), nil
}

// Err returns the error that put the writer in the stopped state, or nil if
// the writer is still active.
func (aw *AsyncWriter) Err() error {
	aw.Lock()
	defer aw.Unlock()

	return aw.err
}

// Wait blocks until all scheduled writes have completed.
func (aw *AsyncWriter) Wait() {
	aw.wg.Wait()
}

// acquire records the presence of a pending write operation. It must be called
// before attempting to schedule a write. It returns an error if the writer has
// been stopped.
func (aw *AsyncWriter) acquire() error {
	aw.Lock()
	defer aw.Unlock()

	// Reject the write operation if the writer is in the stopped state.
	if aw.err != nil {
		return aw.err
	}

	aw.wg.Add(1)
	return nil
}

// release performs the inverse of acquire(). It is called when a scheduled
// write has completed.
func (aw *AsyncWriter) release() {
	aw.wg.Done()
}

// stop puts the writer into the stopped state if it wasn't already so. It
// returns true if the writer was previously active, false otherwise (no-op).
// After being put into the stopped state, the writer rejects new write
// requests, but continues running until all pending requests have completed.
func (aw *AsyncWriter) stop(err error) bool {
	if err == nil {
		panic(fmt.Sprintf("%T.stop() called with err==nil", aw))
	}

	aw.Lock()
	defer aw.Unlock()

	if aw.err != nil {
		// Already stopped.
		return false
	}

	// Don't accept any new writes.
	aw.err = err

	// Close the input channel after all pending writes have completed.
	go func() {
		aw.wg.Wait()
		close(aw.inChan)
	}()

	return true
}

// writeNow writes the given bytes in the current goroutine.
func (aw *AsyncWriter) writeNow(b []byte) error {
	defer aw.release() // Acquired by Write() call in parent goroutine.
	_, err := aw.w.Write(b)
	return err
}

// SyncWriter implements io.Writer. It duplicates output to multiple writers in
// parallel.
type SyncWriter struct {
	aws []*AsyncWriter
}

func NewSyncWriter(ctx context.Context, ws []io.Writer) *SyncWriter {
	var aws []*AsyncWriter
	for _, w := range ws {
		aws = append(aws, NewAsyncWriter(ctx, w))
	}

	return &SyncWriter{
		aws: aws,
	}
}

// Write writes the given bytes to each of the sync writer's constituent
// writers in parallel, then waits for all the writes to complete.
func (sw *SyncWriter) Write(b []byte) (int, error) {
	for _, aw := range sw.aws {
		n, err := aw.Write(b)
		if err != nil {
			return n, err
		}
	}

	// Wait for writes to complete.
	sw.wait()

	return len(b), nil
}

// wait blocks until all scheduled writes have completed.
func (sw *SyncWriter) wait() {
	for _, aw := range sw.aws {
		aw.Wait()
	}
}
