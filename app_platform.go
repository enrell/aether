package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"aether/internal/platform"
	"aether/internal/theme"
	"aether/internal/winapi"
)

// GetPlatform returns the operating system: "linux", "darwin" or "windows".
func (a *App) GetPlatform() string { return runtime.GOOS }

// URLHandlerStatus describes the aether:// registration on Windows.
type URLHandlerStatus struct {
	Supported  bool   `json:"supported"`
	Registered bool   `json:"registered"`
	Current    bool   `json:"current"`
	Command    string `json:"command"`
}

// GetURLHandlerStatus reports whether aether:// links open this executable.
// Only Windows needs explicit registration; Linux registers on startup.
func (a *App) GetURLHandlerStatus() URLHandlerStatus {
	if runtime.GOOS != "windows" {
		return URLHandlerStatus{}
	}
	system, err := winapi.Native()
	if err != nil {
		return URLHandlerStatus{}
	}
	status := URLHandlerStatus{Supported: true}
	command, ok, err := platform.WindowsURLHandler(system.Registry)
	if err != nil || !ok {
		return status
	}
	status.Registered, status.Command = true, command
	if exe, err := os.Executable(); err == nil {
		status.Current = strings.EqualFold(command, platform.URLHandlerCommand(exe))
	}
	return status
}

// RegisterURLHandler makes this executable the current user's aether://
// handler. Windows only; it runs only when the user asks for it.
func (a *App) RegisterURLHandler() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("aether:// links are registered automatically on this platform")
	}
	system, err := winapi.Native()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return platform.RegisterWindowsURLHandler(system.Registry, exe)
}

// UnregisterURLHandler removes Aether's aether:// handler on Windows.
func (a *App) UnregisterURLHandler() error {
	system, err := winapi.Native()
	if err != nil {
		return err
	}
	return platform.UnregisterWindowsURLHandler(system.Registry)
}

// HasWindowsRestore reports whether Aether changed Windows settings that
// RestoreWindowsSettings can put back.
func (a *App) HasWindowsRestore() bool {
	return runtime.GOOS == "windows" && theme.WindowsEnv().HasRestoreRecord()
}

// RestoreWindowsSettings restores the wallpaper, mode, accent colour and the
// Windows Terminal and window manager settings Aether changed.
func (a *App) RestoreWindowsSettings() (string, error) {
	result, err := theme.WindowsEnv().Restore()
	summary := fmt.Sprintf("Restored %d Windows settings", result.Registry)
	if result.Wallpaper != "" {
		summary += ", the wallpaper"
	}
	if n := len(result.Files); n > 0 {
		summary += fmt.Sprintf(" and %d config file(s)", n)
	}
	return summary, err
}

// setWindowsWallpaper changes only the Windows wallpaper.
func setWindowsWallpaper(path string) error {
	return theme.WindowsEnv().ApplyWallpaper(path, nil)
}
