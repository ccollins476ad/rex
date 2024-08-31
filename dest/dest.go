package dest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/badvassal/rex/output"
	"golang.org/x/sys/unix"
)

// Type specifies the broad category of the destination sink.
type Type int

const (
	TypeFD   Type = iota // File descriptor
	TypeFile             // File
	TypeFifo             // Named pipe
	TypeProc             // Child process
)

const (
	unsetType = Type(-1) // Not a valid type.

	defaultPerm = 0644
)

var typeNames = []string{
	TypeFD:   "fd",
	TypeFile: "file",
	TypeFifo: "fifo",
	TypeProc: "proc",
}

var nameTypeMap = map[string]Type{}

func init() {
	for dt, name := range typeNames {
		nameTypeMap[name] = Type(dt)
	}

}

// Dest is a fully self-contained description of a data sink. Use the Open
// method to acquire a corresponding writer for the Dest.
type Dest struct {
	Type        Type
	ID          string
	Perm        uint32
	Args        []string
	NonBlocking bool
	BufSize     int
	Append      bool
	Create      bool
}

// makeDest builds a default-initialized Dest struct. The result is not usable
// for writing data, but it is a suitable initial state for parsing a
// destination specifier string.
func makeDest() Dest {
	return Dest{
		Type: unsetType,
		Perm: defaultPerm,
	}
}

// Open builds a writer associated with the receiver Dest struct. The writer's
// behavior is specified by the Dest's fields.
func (d *Dest) Open() (io.Writer, error) {
	switch d.Type {
	case TypeFD:
		return d.openFD()

	case TypeFile:
		return d.openFile()

	case TypeFifo:
		return d.openFifo()

	case TypeProc:
		return d.openProc()

	default:
		panic(fmt.Sprintf("internal error: invalid dest type: %v", d.Type))
	}
}

// openFD creates a writer for a Dest whose type is TypeFD.
func (d *Dest) openFD() (io.Writer, error) {
	fd, err := strconv.Atoi(d.ID)
	if err != nil {
		return nil, fmt.Errorf("file descriptor has invalid id: have=%s want=<number>: %w", d.ID, err)
	}

	err = d.configureFD(fd)
	if err != nil {
		return nil, err
	}

	return output.NewBestEffortWriter(fd), nil
}

// openFile creates a writer for a Dest whose type is TypeFile.
func (d *Dest) openFile() (io.Writer, error) {
	mode := unix.O_WRONLY
	if d.Create {
		mode |= unix.O_CREAT
	}
	if d.Append {
		mode |= unix.O_APPEND
	} else {
		mode |= unix.O_TRUNC
	}

	fd, err := unix.Open(d.ID, mode, d.Perm)
	if err != nil {
		return nil, err
	}

	err = d.configureFD(fd)
	if err != nil {
		return nil, err
	}

	return output.NewBestEffortWriter(fd), nil
}

// openFifo creates a writer for a Dest whose type is TypeFifo.
func (d *Dest) openFifo() (io.Writer, error) {
	if d.Create {
		err := unix.Mkfifo(d.ID, d.Perm)
		if err != nil && err != syscall.EEXIST {
			return nil, err
		}
	}

	fd, err := unix.Open(d.ID, unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	err = d.configureFD(fd)
	if err != nil {
		return nil, err
	}

	return output.NewBestEffortWriter(fd), nil
}

// openProc creates a writer for a Dest whose type is TypeProc.
func (d *Dest) openProc() (io.Writer, error) {
	cmd := exec.Command(d.ID, d.Args...)

	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// configureFD configures a file descriptor with settings specified in the
// receiver Dest struct's fields.
func (d *Dest) configureFD(fd int) error {
	if d.NonBlocking {
		err := setNonblocking(fd)
		if err != nil {
			return err
		}
	}

	if d.BufSize != 0 {
		err := setBufSize(fd, d.BufSize)
		if err != nil {
			return err
		}
	}

	return nil
}

// setNonblocking puts the given file descriptor in non-blocking mode.
func setNonblocking(fd int) error {
	flags, err := unix.FcntlInt(uintptr(fd), syscall.F_GETFL, 0)
	if err != nil {
		return err
	}

	flags, err = unix.FcntlInt(uintptr(fd), syscall.F_SETFL, flags|unix.O_NONBLOCK)
	if err != nil {
		return err
	}

	return nil
}

// setBufSize configures the pipe buffer size of the given file descriptor.
// When applied to a non-pipe file descriptor, the behavior is unsepcified.
func setBufSize(fd int, size int) error {
	_, err := unix.FcntlInt(uintptr(fd), syscall.F_SETPIPE_SZ, size)
	if err != nil {
		return err
	}
	return nil
}
