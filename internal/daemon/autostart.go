package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/ipc"
)

const autostartTimeout = 5 * time.Second

// Autostart ensures the daemon is running and returns a connected IPC
// client. If the daemon is not reachable, it attempts to start it by
// finding the tmpbufferd binary and forking it.
func Autostart(cfg *config.Config) (*ipc.Conn, error) {
	conn, err := ipc.Dial(cfg.SocketPath, 2*time.Second)
	if err == nil {
		return conn, nil
	}

	daemonPath, err := findDaemonBinary()
	if err != nil {
		return nil, fmt.Errorf("daemon not running and cannot find tmpbufferd binary: %w", err)
	}

	cmd := exec.Command(daemonPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", daemonPath, err)
	}

	conn, err = waitForSocket(cfg.SocketPath, autostartTimeout)
	if err != nil {
		return nil, fmt.Errorf("daemon started but not responding: %w", err)
	}
	return conn, nil
}

// findDaemonBinary locates the tmpbufferd binary, checking PATH first,
// then the directory containing the current executable.
func findDaemonBinary() (string, error) {
	if path, err := exec.LookPath("tmpbufferd"); err == nil {
		return path, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(exe), "tmpbufferd")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("tmpbufferd not found in PATH or next to %s", exe)
}

// waitForSocket polls the UDS path until a dial succeeds or the
// timeout expires.
func waitForSocket(socketPath string, timeout time.Duration) (*ipc.Conn, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := ipc.Dial(socketPath, 500*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout after %v", timeout)
}
