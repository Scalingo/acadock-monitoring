package isgraceful

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type CmdAndOutput struct {
	t      *testing.T
	Cmd    *exec.Cmd
	Output *bytes.Buffer
	pid    int

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
		Output:               new(bytes.Buffer),
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

// WithOutput sets the Output field of the CmdAndOutput struct
func WithOutput(stdout *bytes.Buffer) func(*CmdAndOutput) {
	return func(c *CmdAndOutput) {
		c.Output = stdout
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

	// get pid from pid file
	if c.pidFile != "" {
		c.pid = c.readPidFile()
	}

	err := c.findProcess().Signal(signal)
	if err != nil {
		c.t.Fatalf("send signal %v: %v", signal, err)
	}
}

// Start starts the process
func (c *CmdAndOutput) Start() {
	c.t.Helper()
	c.Cmd.Stdout = c.Output
	c.Cmd.Stderr = c.Output
	err := c.Cmd.Start()
	if err != nil {
		c.t.Fatalf("failed to start process: %v", err)
	}

	// Get the pid
	c.pid = c.Cmd.Process.Pid

	// Wait for a short duration to allow the child process to start
	c.IsRunningAfter(c.startWaitDuration)
}

// Stop stops the process
func (c *CmdAndOutput) Stop() {
	// Wait for the command (this must be called after start)
	go func() {
		_ = c.Cmd.Wait()
	}()

	p := c.findProcess()

	// send signal to pid process
	err := syscall.Kill(p.Pid, syscall.SIGINT)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		c.t.Logf("kill process: %v", err)
	}

	c.IsStoppedAfter(c.shutdownWaitDuration)
}

// IsStoppedAfter checks if the process is stopped after a certain duration
func (c *CmdAndOutput) IsStoppedAfter(timeout time.Duration) {
	// c.t.Helper()

	// Has any process started
	require.NotNilf(c.t, c.Cmd.Process, "process %v hasn't started", c.Cmd)

	time.Sleep(timeout)

	// Nil if the process running
	require.NotNilf(c.t, c.findProcess(), "process %v was up after %v, output: \n\n%v", c.Cmd, timeout, c.Output.String())
}

// IsRunningAfter checks if the process is running after a certain duration
func (c *CmdAndOutput) IsRunningAfter(timeout time.Duration) {
	c.t.Helper()

	// Has any process started
	require.NotNilf(c.t, c.Cmd.Process, "process %v hasn't started", c.Cmd)

	time.Sleep(timeout)

	var processState bool
	if c.Cmd.ProcessState != nil {
		processState = c.Cmd.ProcessState.Success()
	}

	// Nil if the process running
	require.NoErrorf(c.t, c.findProcess().Signal(syscall.Signal(0)),
		"process %v is dead after %v, status: %v, output: \n\n%v", c.Cmd.Args, timeout, processState, c.Output.String())
}

func (c *CmdAndOutput) readPidFile() int {
	// c.t.Helper()
	data, err := os.ReadFile(c.pidFile)
	require.NoError(c.t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(c.t, err)
	c.t.Logf("pid: %v", pid)
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
