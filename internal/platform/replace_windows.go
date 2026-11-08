//go:build windows

package platform

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// replaceFile renames from over to. Windows refuses to replace a file that
// another process has open (an editor, a terminal reloading its settings,
// Aether's own theme watcher), so sharing violations are retried briefly.
func replaceFile(from, to string) error {
	var err error
	for delay := 10 * time.Millisecond; delay <= 640*time.Millisecond; delay *= 2 {
		if err = os.Rename(from, to); err == nil || !isSharingError(err) {
			return err
		}
		time.Sleep(delay)
	}
	return err
}

func isSharingError(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED) ||
		errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
