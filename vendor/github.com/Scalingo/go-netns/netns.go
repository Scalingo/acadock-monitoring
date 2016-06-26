package netns

import (
	"errors"
	"os"
	"syscall"
)

var (
	NamespaceCloseError = errors.New("namespace close")
)

type NetworkNsHandler struct {
	ns     *os.File
	closed bool
}

func Setns(pid string) (*NetworkNsHandler, error) {
	return SetnsFromProcDir("/proc/" + pid)
}

func SetnsFromProcDir(procDir string) (*NetworkNsHandler, error) {
	var err error
	h := &NetworkNsHandler{}
	h.ns, err = os.Open(procDir + "/ns/net")
	if err != nil {
		return nil, err
	}

	err = setns(h.ns.Fd())
	if err != nil {
		h.ns.Close()
		return nil, err
	}

	return h, nil
}

func setns(fd uintptr) error {
	_, _, err := syscall.Syscall(308, fd, uintptr(0), uintptr(0))
	if uintptr(err) == 0 {
		return nil
	}
	return err
}

func (h *NetworkNsHandler) Close() error {
	h.closed = true
	return h.ns.Close()
}
