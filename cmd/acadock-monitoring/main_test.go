package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Scalingo/acadock-monitoring/cmd/acadock-monitoring/isgraceful"

	"github.com/stretchr/testify/require"
)

// getCmd returns a command to run the main.go file
func getCmd(t *testing.T, tmpdir string) *exec.Cmd {
	binaryPath := tmpdir + "/main"

	// Clean the temporary directory
	_ = os.Remove(binaryPath)

	// Build main.go
	err := exec.Command("go", "build", "-o", binaryPath, "main.go").Run()
	require.NoError(t, err)

	return exec.Command(binaryPath)
}

func TestService_Shutdown(t *testing.T) {
	// Set some environment variables
	upgradeTimeout := time.Millisecond * 200
	shutdownTimeout := time.Millisecond * 100
	t.Setenv("GRACEFUL_UPGRADE_TIMEOUT", upgradeTimeout.String())
	t.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", shutdownTimeout.String())

	for _, s := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run("Signal "+s.String(), func(t *testing.T) {
			// Create a temporary directory for the built command
			tmpdir, err := os.MkdirTemp(t.TempDir(), "")
			require.NoError(t, err)
			defer require.NoError(t, os.RemoveAll(tmpdir))

			// Pid environment variable is test specific
			pidFile := tmpdir + "/main.pid"
			t.Setenv("GRACEFUL_PID_FILE", pidFile)

			// Configure isGraceful
			isGraceful := isgraceful.NewCmdAndOutput(t,
				isgraceful.WithCmd(getCmd(t, tmpdir)),
				isgraceful.WithUpgradeWaitDuration(upgradeTimeout),
				isgraceful.WithShutdownWaitDuration(shutdownTimeout),
				isgraceful.WithPidFile(pidFile),
			)

			// Start the command
			isGraceful.Start()
			defer isGraceful.Stop()

			// Send the signal
			isGraceful.Signal(s)
			isGraceful.IsStoppedAfter(shutdownTimeout)

			// Check the output
			output := isGraceful.Output.String()
			t.Log(output)
			require.Equal(t, 1, strings.Count(output, "parent exited"))
			require.Zero(t, strings.Count(output, "upgrade requested"))
			require.Zero(t, strings.Count(output, "upgrade failed"))
		})
	}
}

func TestService_Restart(t *testing.T) {
	// Set some environment variables
	upgradeTimeout := time.Millisecond * 200
	shutdownTimeout := time.Millisecond * 100
	t.Setenv("GRACEFUL_UPGRADE_TIMEOUT", upgradeTimeout.String())
	t.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", shutdownTimeout.String())

	// Create a temporary directory for the built command
	tmpdir, err := os.MkdirTemp(t.TempDir(), "")
	require.NoError(t, err)
	defer require.NoError(t, os.RemoveAll(tmpdir))

	// Pid environment variable is test specific
	pidFile := tmpdir + "/main.pid"
	t.Setenv("GRACEFUL_PID_FILE", pidFile)

	// Configure isGraceful
	isGraceful := isgraceful.NewCmdAndOutput(t,
		isgraceful.WithCmd(getCmd(t, tmpdir)),
		isgraceful.WithUpgradeWaitDuration(upgradeTimeout),
		isgraceful.WithShutdownWaitDuration(shutdownTimeout),
		isgraceful.WithPidFile(pidFile),
	)

	// Start the command
	isGraceful.Start()
	defer isGraceful.Stop()

	// Send restart signal
	isGraceful.Signal(syscall.SIGHUP)
	isGraceful.IsRunningAfter(upgradeTimeout)

	// Check the output
	output := isGraceful.Output.String()
	t.Log(output)
	require.Equal(t, 1, strings.Count(output, "upgrade requested"))
	require.Zero(t, strings.Count(output, "upgrade failed"))
}

func TestService_Restart_Twice(t *testing.T) {
	// Set some environment variables
	upgradeTimeout := time.Millisecond * 200
	shutdownTimeout := time.Millisecond * 100
	t.Setenv("GRACEFUL_UPGRADE_TIMEOUT", upgradeTimeout.String())
	t.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", shutdownTimeout.String())

	// Create a temporary directory for the built command
	tmpdir, err := os.MkdirTemp(t.TempDir(), "")
	require.NoError(t, err)
	defer require.NoError(t, os.RemoveAll(tmpdir))

	// Pid environment variable is test specific
	pidFile := tmpdir + "/main.pid"
	t.Setenv("GRACEFUL_PID_FILE", pidFile)

	// Configure isGraceful
	isGraceful := isgraceful.NewCmdAndOutput(t,
		isgraceful.WithCmd(getCmd(t, tmpdir)),
		isgraceful.WithUpgradeWaitDuration(upgradeTimeout),
		isgraceful.WithShutdownWaitDuration(shutdownTimeout),
		isgraceful.WithPidFile(pidFile),
	)

	// Start the command
	isGraceful.Start()
	defer isGraceful.Stop()

	// Send restart signal
	isGraceful.Signal(syscall.SIGHUP)
	isGraceful.IsRunningAfter(upgradeTimeout)

	// Check the output
	output := isGraceful.Output.String()
	t.Log(output)
	require.Equal(t, 1, strings.Count(output, "upgrade requested"))
	require.Zero(t, strings.Count(output, "upgrade failed"))

	// Send restart signal
	isGraceful.Signal(syscall.SIGHUP)
	isGraceful.IsRunningAfter(upgradeTimeout)

	// Check the output
	output = isGraceful.Output.String()
	t.Log(output)
	require.Equal(t, 2, strings.Count(output, "upgrade requested"))
	require.Zero(t, strings.Count(output, "upgrade failed"))
}
