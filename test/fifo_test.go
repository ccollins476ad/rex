package test

import (
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/badvassal/rex/test/testutil"
	"github.com/tj/assert"
)

func tempFifoFilename() string {
	return testutil.TempFilename("rextest-fifo-")
}

// One fifo; 1MB
func TestFifo1(t *testing.T) {
	filename := tempFifoFilename()
	args := []string{fmt.Sprintf("type=fifo,id=%s,create", filename)}

	rexCmd, err := testutil.StartRex(args)
	assert.NoError(t, err)
	defer os.Remove(filename)

	// Need to open the fifo before rex writes to it, otherwise os will discard
	// data.
	f, err := testutil.Wait1SForFileThenOpen(filename)
	assert.NoError(t, err)

	lhs := testutil.RandBytes(testutil.MB)
	go func() {
		defer rexCmd.Stdin.Close()
		_, err := rexCmd.Stdin.Write(lhs)
		assert.NoError(t, err)
	}()

	rhs, err := io.ReadAll(f)
	assert.NoError(t, err)

	assert.Equal(t, lhs, rhs)
}

// Two fifos; 1MB bytes
func TestFifo2(t *testing.T) {
	filename1 := tempFifoFilename()
	filename2 := tempFifoFilename()
	args := []string{
		fmt.Sprintf("type=fifo,id=%s,create", filename1),
		fmt.Sprintf("type=fifo,id=%s,create", filename2),
	}

	rexCmd, err := testutil.StartRex(args)
	assert.NoError(t, err)
	defer os.Remove(filename1)
	defer os.Remove(filename2)

	// Need to open the fifos before rex writes to them, otherwise os will
	// discard data.
	f1, err := testutil.Wait1SForFileThenOpen(filename1)
	assert.NoError(t, err)
	f2, err := testutil.Wait1SForFileThenOpen(filename2)
	assert.NoError(t, err)

	lhs := testutil.RandBytes(testutil.MB)
	go func() {
		defer rexCmd.Stdin.Close()
		_, err := rexCmd.Stdin.Write(lhs)
		assert.NoError(t, err)
	}()

	var (
		wg   sync.WaitGroup
		rhs1 []byte
		rhs2 []byte
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		rhs1, err = io.ReadAll(f1)
		assert.NoError(t, err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		rhs2, err = io.ReadAll(f2)
		assert.NoError(t, err)
	}()

	wg.Wait()

	assert.Equal(t, lhs, rhs1)
	assert.Equal(t, lhs, rhs2)
}

// One fifo,nonblocking,bufsize=16KB; 1MB
func TestFifoNonblocking(t *testing.T) {
	filename := tempFifoFilename()
	bufSize := 16 * testutil.KB
	args := []string{fmt.Sprintf("type=fifo,id=%s,create,nonblocking,bufsize=%d", filename, bufSize)}

	rexCmd, err := testutil.StartRex(args)
	assert.NoError(t, err)
	defer os.Remove(filename)

	// Need to open the fifo before rex writes to it, otherwise os will discard
	// data.
	f, err := testutil.Wait1SForFileThenOpen(filename)
	assert.NoError(t, err)

	// Blocking write of all 1MB of data.
	lhs := testutil.RandBytes(testutil.MB)
	_, err = rexCmd.Stdin.Write(lhs)
	assert.NoError(t, err)
	rexCmd.Stdin.Close()

	// Ensure first 16KB is readable and rest was discarded.
	rhs, err := io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, lhs[:bufSize], rhs)
}
