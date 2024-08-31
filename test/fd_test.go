package test

import (
	"io"
	"strconv"
	"testing"

	"github.com/badvassal/rex/test/testutil"
	"github.com/tj/assert"
)

func testFDNoDupOnce(t *testing.T, readBufSize int, writeSize int, destArgs []string, expStdout bool, expStderr bool) {
	args := append([]string{"-b", strconv.Itoa(readBufSize)}, destArgs...)

	rexCmd, err := testutil.StartRex(args)
	assert.NoError(t, err)

	testStr := testutil.RandString(writeSize)

	rexCmd.Stdin.Write([]byte(testStr))
	rexCmd.Stdin.Close()

	checkStream := func(r io.ReadCloser, expFull bool) {
		b, err := io.ReadAll(r)
		assert.NoError(t, err)
		if expFull {
			assert.Equal(t, []byte(testStr), b)
		} else {
			assert.Equal(t, []byte{}, b)
		}
	}

	checkStream(rexCmd.Stdout, expStdout)
	checkStream(rexCmd.Stderr, expStderr)

	err = rexCmd.Cmd.Wait()
	assert.NoError(t, err)
}

// Output to type=fd only; each fd specified only once.
func TestFDNoDup(t *testing.T) {
	// read buf > write size
	testFDNoDupOnce(t, 100, 10, []string{"type=fd,id=1"}, true, false)
	testFDNoDupOnce(t, 100, 10, []string{"type=fd,id=2"}, false, true)
	testFDNoDupOnce(t, 100, 10, []string{"type=fd,id=1", "type=fd,id=2"}, true, true)

	// read buf = write size
	testFDNoDupOnce(t, 100, 100, []string{"type=fd,id=1"}, true, false)
	testFDNoDupOnce(t, 100, 100, []string{"type=fd,id=2"}, false, true)
	testFDNoDupOnce(t, 100, 100, []string{"type=fd,id=1", "type=fd,id=2"}, true, true)

	// read buf < write size
	testFDNoDupOnce(t, 1, 100, []string{"type=fd,id=1"}, true, false)
	testFDNoDupOnce(t, 1, 100, []string{"type=fd,id=2"}, false, true)
	testFDNoDupOnce(t, 1, 100, []string{"type=fd,id=1", "type=fd,id=2"}, true, true)
}

func testFDDupOnce(t *testing.T, readBufSize int, writeSize int, numDups int) {
	args := []string{"-b", strconv.Itoa(readBufSize)}
	for i := 0; i < numDups; i++ {
		args = append(args, "type=fd,id=1")
	}

	rexCmd, err := testutil.StartRex(args)
	assert.NoError(t, err)

	testStr := testutil.RandString(1024)

	rexCmd.Stdin.Write([]byte(testStr))
	rexCmd.Stdin.Close()

	b, err := io.ReadAll(rexCmd.Stdout)
	assert.NoError(t, err)

	assert.Equal(t, expDupOutput(testStr, readBufSize, numDups), string(b))
}

// Calculates expected output for >1 stdout df specifier.
func expDupOutput(in string, readBufSize int, numDups int) string {
	out := make([]byte, len(in)*numDups)

	// Read readBufSize bytes from input, write it numDups times. Repeat until
	// all input consumed.
	var inOff int
	var outOff int
	for inOff < len(in) {
		inRem := len(in) - inOff
		chunkSz := readBufSize
		if chunkSz > inRem {
			chunkSz = inRem
		}

		for i := 0; i < numDups; i++ {
			copy(out[outOff:], in[inOff:])
			outOff += chunkSz
		}
		inOff += chunkSz
	}

	return string(out)
}

// Output to type=fd,id=1; fd specified multiple times.
func TestFDDup(t *testing.T) {
	testFDDupOnce(t, 100, 100, 2)
	testFDDupOnce(t, 100, 100, 10)
	testFDDupOnce(t, 10, 100, 10)
}
