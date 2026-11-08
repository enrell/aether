//go:build !windows

package ipc

import (
	"os"
	"path/filepath"
	"testing"
)

func testSocketPath(t *testing.T) string {
	t.Helper()
	// Unix socket paths are limited to ~100 bytes, so avoid long TempDirs.
	dir, err := os.MkdirTemp("", "aether-ipc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "aether.sock")
}
