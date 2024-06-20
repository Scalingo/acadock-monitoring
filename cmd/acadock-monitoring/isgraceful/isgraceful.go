package isgraceful

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type CmdAndOutput struct {
	t   *testing.T
	Cmd *exec.Cmd
	pid int

	waitGroup sync.WaitGroup

	output    *bytes.Buffer
	outputMu  sync.Mutex
	oldStdout io.Writer
	oldStderr io.Writer

	// shutdownWaitDuration is the duration which is waited for all connections to stop
	shutdownWaitDuration time.Duration

	// startWaitDuration is the duration to wait for a child process to start
	startWaitDuration time.Duration

	// upgradeWaitDuration is the duration the old process is waiting for
	// connection to close when a graceful restart has been ordered.
	upgradeWaitDuration time.Duration

	// pidFile tracks the pid of the last child among the chain of graceful restart
	pidFile string
}

// NewCmdAndOutput creates a new CmdAndOutput struct using the functional options pattern
func NewCmdAndOutput(t *testing.T, options ...func(*CmdAndOutput)) *CmdAndOutput {
	t.Helper()
	c := &CmdAndOutput{
		t:                    t,
		output:               new(bytes.Buffer),
		startWaitDuration:    100 * time.Millisecond,
		upgradeWaitDuration:  30 * time.Second,
		shutdownWaitDuration: 60 * time.Second,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// WithCmd sets the Cmd field of the CmdAndOutput struct
func WithCmd(cmd *exec.Cmd) func(*CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.Cmd = cmd
	}
}

// WithOutput sets the output field of the CmdAndOutput struct
func WithOutput(stdout *bytes.Buffer) func(*CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.output = stdout
	}
}

// WithStartWaitDuration sets the duration to wait for a child process to start
func WithStartWaitDuration(duration time.Duration) func(output *CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.startWaitDuration = duration
	}
}

// WithPidFile sets the pidFile field of the CmdAndOutput struct
func WithPidFile(pidFile string) func(*CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.pidFile = pidFile
	}
}

// WithUpgradeWaitDuration sets the duration the old process is waiting for
func WithUpgradeWaitDuration(duration time.Duration) func(output *CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.upgradeWaitDuration = duration
	}
}

// WithShutdownWaitDuration sets the duration which is waited for all connections to stop
func WithShutdownWaitDuration(duration time.Duration) func(output *CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.shutdownWaitDuration = duration
	}
}

// Signal sends a signal to the process
func (c *CmdAndOutput) Signal(signal os.Signal) {
	c.t.Helper()

	err := c.findProcess().Signal(signal)
	if err != nil {
		c.t.Fatalf("send signal %v: %v", signal, err)
	}
}

// Start starts the process
func (c *CmdAndOutput) Start() {
	c.t.Helper()

	c.oldStdout = c.Cmd.Stdout
	c.oldStderr = c.Cmd.Stderr
	r, w, _ := os.Pipe()
	c.Cmd.Stdout = w
	c.Cmd.Stderr = w

	// Read from pipe and append to buffer with locking
	go func() {
		b := make([]byte, 1024)
		for {
			n, err := r.Read(b)
			if n > 0 {
				c.outputMu.Lock()
				c.output.Write(b[:n])
				c.outputMu.Unlock()
			}
			if err != nil {
				break
			}
		}
	}()

	err := c.Cmd.Start()
	if err != nil {
		c.t.Fatalf("failed to start process: %v", err)
	}

	// Get the pid
	c.pid = c.Cmd.Process.Pid

	// Write the pid to the pid file
	if c.pidFile != "" {
		err := os.WriteFile(c.pidFile, []byte(strconv.Itoa(c.pid)), 0600)
		require.NoError(c.t, err)
	}

	// Wait for a short duration to allow the child process to start
	time.Sleep(c.startWaitDuration)
}

// Stop stops the process
func (c *CmdAndOutput) Stop() {
	// Wait for all (isRunningAfter / isStoppedAfter) operations to finish
	c.waitGroup.Wait()

	// send signal to parent process
	err := syscall.Kill(c.Cmd.Process.Pid, syscall.SIGTERM)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		c.t.Logf("kill process: %v", err)
	}

	// send signal to pid process
	err = syscall.Kill(c.pid, syscall.SIGTERM)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		c.t.Logf("kill process: %v", err)
	}

	// Wait for the parent or child processes to finish
	c.IsStoppedAfter(c.shutdownWaitDuration)
}

// IsRunningAfter checks if the process is running after a certain duration
func (c *CmdAndOutput) IsRunningAfter(timeout time.Duration) {
	c.t.Helper()
	c.CheckProcessAfter(timeout, true)
}

// IsRunningAfterAsync checks if the process is running after a certain duration, asynchronously
func (c *CmdAndOutput) IsRunningAfterAsync(timeout time.Duration) {
	c.t.Helper()
	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		c.CheckProcessAfter(timeout, true)
	}()
}

// IsStoppedAfter checks if the process is stopped after a certain duration
func (c *CmdAndOutput) IsStoppedAfter(timeout time.Duration) {
	c.t.Helper()
	c.CheckProcessAfter(timeout, false)
}

// IsStoppedAfterAsync checks if the process is stopped after a certain duration, asynchronously
func (c *CmdAndOutput) IsStoppedAfterAsync(timeout time.Duration) {
	c.t.Helper()
	c.waitGroup.Add(1)
	go func() {
		defer c.waitGroup.Done()
		c.CheckProcessAfter(timeout, false)
	}()
}

// CheckProcessAfter checks the process is running after a certain duration
func (c *CmdAndOutput) CheckProcessAfter(timeout time.Duration, shouldBeAlive bool) {
	c.t.Helper()

	// Has any process started
	require.NotNilf(c.t, c.Cmd.Process, "process %v hasn't started", c.Cmd)

	if shouldBeAlive {
		// Wait and then search for the process (parent or child)
		time.Sleep(timeout)
		p := c.findProcess()
		require.NoErrorf(c.t, p.Signal(syscall.Signal(0)), "process %v is dead after %v", c.pid, timeout)
	} else {
		// Race between the timer and the process
		w := make(chan *os.ProcessState)
		go func() {
			processState, _ := c.findProcess().Wait()
			w <- processState
			close(w)
		}()

		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-timer.C:
			c.t.Errorf("process %v was up after %v", c.pid, timeout)
		case <-w:
		}
	}
}

// GetOutput returns the output of the process
func (c *CmdAndOutput) GetOutput() string {
	c.waitGroup.Wait()

	c.outputMu.Lock()
	defer c.outputMu.Unlock()
	return c.output.String()
}

func (c *CmdAndOutput) readPidFile() int {
	c.t.Helper()
	data, err := os.ReadFile(c.pidFile)
	require.NoError(c.t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(c.t, err)
	return pid
}

func (c *CmdAndOutput) findProcess() *os.Process {
	// get pid from pid file
	if c.pidFile != "" {
		c.pid = c.readPidFile()
	}

	p, err := os.FindProcess(c.pid)
	require.NoError(c.t, err)
	return p
}
