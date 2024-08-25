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

type Type int

const (
	TypeFD   Type = iota // File descriptor
	TypeFile             // File
	TypeFifo             // Named pipe
	TypeProc             // Child process
)

const (
	unsetType   = Type(-1) // Not a valid type.
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

func makeDest() Dest {
	return Dest{
		Type: unsetType,
		Perm: defaultPerm,
	}
}

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

func setBufSize(fd int, size int) error {
	_, err := unix.FcntlInt(uintptr(fd), syscall.F_SETPIPE_SZ, size)
	if err != nil {
		return err
	}
	return nil
}
