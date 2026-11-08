//go:build !windows

package ipc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"aether/internal/platform"
)

// DefaultSocketPath returns the path for the Aether IPC socket.
func DefaultSocketPath() string {
	return filepath.Join(platform.ConfigDir(), "aether.sock")
}

// listen opens the Unix socket, restricts it to the owner and writes the PID
// file used for stale-socket detection.
func listen(sockPath string) (net.Listener, func(), error) {
	if err := cleanStaleSocket(sockPath); err != nil {
		return nil, nil, err
	}

	if err := os.MkdirAll(filepath.Dir(sockPath), 0700); err != nil {
		return nil, nil, fmt.Errorf("ipc: mkdir: %w", err)
	}

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, nil, fmt.Errorf("ipc: listen: %w", err)
	}

	if err := os.Chmod(sockPath, 0600); err != nil {
		ln.Close()
		os.Remove(sockPath)
		return nil, nil, fmt.Errorf("ipc: chmod: %w", err)
	}

	pidPath := sockPath + ".pid"
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		ln.Close()
		os.Remove(sockPath)
		return nil, nil, fmt.Errorf("ipc: write pid: %w", err)
	}
	cleanup := func() {
		os.Remove(sockPath)
		os.Remove(pidPath)
	}
	return ln, cleanup, nil
}

func dial(sockPath string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", sockPath, timeout)
}

func notRunning(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED)
}

// cleanStaleSocket removes a leftover socket from a crashed instance.
func cleanStaleSocket(sockPath string) error {
	pidPath := sockPath + ".pid"
	pidData, err := os.ReadFile(pidPath)
	if err != nil {
		// No PID file — remove orphan socket if it exists
		os.Remove(sockPath)
		return nil
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		os.Remove(pidPath)
		os.Remove(sockPath)
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		os.Remove(sockPath)
		return nil
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		// Process is dead — clean up stale files
		os.Remove(pidPath)
		os.Remove(sockPath)
		return nil
	}

	// Process is still running
	return fmt.Errorf("ipc: aether is already running (pid %d)", pid)
}
