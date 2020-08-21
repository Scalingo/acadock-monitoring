package procfs

import (
	"io"
	"os"
)

type testFileSystem struct {
	file string
}

func (f testFileSystem) Open(filename string) (io.ReadCloser, error) {
	return os.Open(f.file)
}
