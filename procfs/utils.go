package procfs

import (
	"context"
	"io"
	"os"

	"github.com/tklauser/go-sysconf"

	"github.com/Scalingo/go-utils/errors/v3"
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

type FileSystem struct {
	ctx context.Context
}

func NewFileSystem(ctx context.Context) FileSystem {
	return FileSystem{
		ctx: ctx,
	}
}

func (fs FileSystem) Open(filename string) (io.ReadCloser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(fs.ctx, err, "open file")
	}
	return file, nil
}
