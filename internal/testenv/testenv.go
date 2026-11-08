// Package testenv isolates tests from the real user profile on every
// platform. Linux resolves Aether's directories from HOME and XDG_*; Windows
// uses USERPROFILE, APPDATA and LOCALAPPDATA, so setting HOME alone would let
// a Windows test write to the real profile.
package testenv

import (
	"path/filepath"
	"runtime"
	"testing"
)

// Home points HOME, the XDG base directories and the Windows profile
// folders at home and returns it. An empty home uses t.TempDir().
func Home(t testing.TB, home string) string {
	t.Helper()
	if home == "" {
		home = t.TempDir()
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	WindowsProfile(t, home)
	return home
}

// WindowsProfile points the Windows profile folders at home without touching
// HOME or XDG_*, for tests that set those themselves.
func WindowsProfile(t testing.TB, home string) {
	t.Helper()
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
}

// SkipOnWindows skips a test that checks Unix-only behaviour (permission
// bits, symlinks, shell scripts, Omarchy, XDG icon themes).
func SkipOnWindows(t testing.TB, reason string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix only: " + reason)
	}
}
