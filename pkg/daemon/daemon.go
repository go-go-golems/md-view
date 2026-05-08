package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	AppName = "md-view"
)

// StateDir returns the XDG state directory for md-view.
//
// #nosec G703 -- XDG_STATE_HOME is a standard env var, not untrusted input
func StateDir() (string, error) {
	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		xdgState = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(xdgState, AppName)
	return dir, os.MkdirAll(dir, 0755)
}

func pidFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".pid"), nil
}

func portFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".port"), nil
}

func socketFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName+".sock"), nil
}

// WritePID writes the current process PID to the state file.
func WritePID() error {
	path, err := pidFile()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// ReadPID reads the daemon PID from the state file.
func ReadPID() (int, error) {
	path, err := pidFile()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}
	return pid, nil
}

// WritePort writes the HTTP port to the state file.
func WritePort(port int) error {
	path, err := portFile()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(port)), 0644)
}

// ReadPort reads the HTTP port from the state file.
func ReadPort() (int, error) {
	path, err := portFile()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid port file content: %w", err)
	}
	return port, nil
}

// SocketPath returns the Unix socket path.
func SocketPath() (string, error) {
	return socketFile()
}

// IsAlive checks if a process with the given PID is running.
func IsAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// DaemonStatus represents the current state of the daemon.
type DaemonStatus struct {
	Running    bool
	PID        int
	Port       int
	SocketPath string
	StartTime  time.Time
}

// GetStatus returns the current daemon status.
func GetStatus() (*DaemonStatus, error) {
	status := &DaemonStatus{}

	sockPath, err := SocketPath()
	if err != nil {
		return status, nil
	}
	status.SocketPath = sockPath

	pid, err := ReadPID()
	if err != nil {
		return status, nil
	}
	status.PID = pid

	if !IsAlive(pid) {
		// Stale PID file — clean up
		_ = Cleanup()
		return status, nil
	}
	status.Running = true

	port, err := ReadPort()
	if err == nil {
		status.Port = port
	}

	// Use PID file modification time as approximate start time
	pidPath, _ := pidFile()
	if info, err := os.Stat(pidPath); err == nil {
		status.StartTime = info.ModTime()
	}

	return status, nil
}

// Cleanup removes all state files (PID, port, socket).
func Cleanup() error {
	for _, fn := range []func() (string, error){pidFile, portFile, socketFile} {
		path, err := fn()
		if err != nil {
			continue
		}
		_ = os.Remove(path)
	}
	return nil
}

// Stop sends SIGTERM to the daemon process.
func Stop() error {
	pid, err := ReadPID()
	if err != nil {
		return fmt.Errorf("cannot read PID: %w", err)
	}
	if !IsAlive(pid) {
		_ = Cleanup()
		return fmt.Errorf("daemon not running (stale PID %d)", pid)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("cannot find process: %w", err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("cannot send SIGTERM: %w", err)
	}

	// Wait a bit for cleanup
	for i := 0; i < 50; i++ {
		if !IsAlive(pid) {
			_ = Cleanup()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill
	_ = process.Signal(syscall.SIGKILL)
	_ = Cleanup()
	return nil
}
