package procfs

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/tklauser/go-sysconf"
)

// Collection of utils needed to mock real system interactions during tests.

var _ Sysconf = NewSysconf()

type Sysconf interface {
	ClockTick() (int64, error)
}

type SysconfReader struct{}

func NewSysconf() SysconfReader {
	return SysconfReader{}
}

func (SysconfReader) ClockTick() (int64, error) {
	return sysconf.Sysconf(sysconf.SC_CLK_TCK)
}

type FS interface {
	Open(string) (io.ReadCloser, error)
}

type FileSystem struct{}

func NewFileSystem() FileSystem {
	return FileSystem{}
}

func (FileSystem) Open(filename string) (io.ReadCloser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "fail to open file")
	}
	return file, nil
}
