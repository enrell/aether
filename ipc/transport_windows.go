//go:build windows

package ipc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// DefaultSocketPath returns the named pipe of the current user's Aether.
// Pipes live in a machine-wide namespace, so the name carries the user SID.
func DefaultSocketPath() string {
	return `\\.\pipe\aether-` + pipeUser()
}

func pipeUser() string {
	if sid, err := currentUserSID(); err == nil {
		return sid
	}
	return strings.NewReplacer(`\`, "-", "/", "-").Replace(os.Getenv("USERNAME"))
}

func currentUserSID() (string, error) {
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String(), nil
}

// listen creates the named pipe. Only the current user (and SYSTEM) may open
// it, and creation fails if another process already owns the name.
func listen(pipePath string) (net.Listener, func(), error) {
	sid, err := currentUserSID()
	if err != nil {
		return nil, nil, fmt.Errorf("ipc: current user: %w", err)
	}
	ln, err := winio.ListenPipe(pipePath, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;" + sid + ")(A;;GA;;;SY)",
		InputBufferSize:    64 * 1024,
		OutputBufferSize:   64 * 1024,
	})
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_PIPE_BUSY) {
			return nil, nil, fmt.Errorf("ipc: aether is already running (%s)", pipePath)
		}
		return nil, nil, fmt.Errorf("ipc: listen: %w", err)
	}
	return ln, func() {}, nil
}

func dial(pipePath string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(pipePath, &timeout)
}

func notRunning(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND)
}
