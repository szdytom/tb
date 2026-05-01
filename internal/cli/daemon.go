package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/szdytom/tb/internal/config"
	"github.com/szdytom/tb/internal/daemon"
	"github.com/szdytom/tb/internal/ipc"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the tmpbuffer daemon",
	}
	cmd.AddCommand(
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonStatusCmd(),
	)
	return cmd
}

func newDaemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemonStart()
		},
	}
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemonStop()
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemonStatus()
		},
	}
}

func runDaemonStart() error {
	cfg := config.Default()

	// Check if already running.
	conn, err := ipc.Dial(cfg.SocketPath, 500*time.Millisecond)
	if err == nil {
		conn.Close()
		fmt.Fprintln(os.Stderr, "Daemon is already running")
		return nil
	}

	daemonPath, err := daemon.FindDaemonBinary()
	if err != nil {
		printError(err.Error())
		return err
	}

	daemonArgs := []string{}
	if configFile != "" {
		daemonArgs = append(daemonArgs, "-c", configFile)
	}
	cmd := exec.Command(daemonPath, daemonArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		printError("start daemon: " + err.Error())
		return err
	}

	// Wait for the socket to appear.
	if _, err := waitForSocket(cfg.SocketPath, 5*time.Second); err != nil {
		printError("daemon started but not responding")
		return err
	}

	fmt.Fprintf(os.Stderr, "Daemon started (PID %d)\n", cmd.Process.Pid)
	return nil
}

func runDaemonStop() error {
	cfg := config.Default()

	pidData, err := os.ReadFile(cfg.PidFilePath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Daemon is not running")
		return nil
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		printError("invalid PID file")
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		// Process not found; clean up PID file.
		os.Remove(cfg.PidFilePath())
		fmt.Fprintln(os.Stderr, "Daemon is not running")
		return nil
	}

	if err := proc.Signal(os.Interrupt); err != nil {
		printError("stop daemon: " + err.Error())
		return err
	}

	// Wait for the socket to disappear.
	for i := 0; i < 30; i++ {
		if _, err := os.Stat(cfg.SocketPath); os.IsNotExist(err) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "Daemon stopped (PID %d)\n", pid)
	return nil
}

func runDaemonStatus() error {
	cfg := config.Default()

	conn, err := ipc.Dial(cfg.SocketPath, 500*time.Millisecond)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Daemon is not running")
		return fmt.Errorf("not running")
	}
	defer conn.Close()

	// Use a client to ping.
	client := &Client{conn: conn, nextID: 1}
	if err := client.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "Daemon is not running")
		return fmt.Errorf("not running")
	}

	// Try to read PID file.
	pidStr := "unknown"
	if data, err := os.ReadFile(cfg.PidFilePath()); err == nil {
		pidStr = strings.TrimSpace(string(data))
	}

	fmt.Fprintf(os.Stderr, "Daemon is running (PID %s)\n", pidStr)
	return nil
}

// waitForSocket polls the UDS path until a dial succeeds or the timeout
// expires. This is a copy of the logic in daemon/autostart.go to avoid
// circular dependencies.
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
