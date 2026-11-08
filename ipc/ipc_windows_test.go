//go:build windows

package ipc

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func testSocketPath(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf(`\\.\pipe\aether-test-%d`, time.Now().UnixNano())
}

func TestDefaultSocketPathIsPerUserPipe(t *testing.T) {
	path := DefaultSocketPath()
	if !strings.HasPrefix(path, `\\.\pipe\aether-S-1-5-`) {
		t.Errorf("DefaultSocketPath() = %q, want a pipe named after the user SID", path)
	}
}
