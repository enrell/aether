// Package wintheme applies Aether themes to Windows: the wallpaper, light/dark
// mode and accent colour, Windows Terminal, the LeopardWM and komorebi window
// managers, and Windows installs of cross-platform apps. Every target reads a
// rendered Aether template, changes only the settings it owns, keeps a backup
// or a restore record, and reloads the application where it can.
//
// The package has no build tags: Win32 and registry access go through
// winapi.System and external commands through Env.Run, so the logic is tested
// with fakes on every platform.
package wintheme

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"aether/internal/winapi"
)

// Aether app names of the Windows-only targets.
const (
	AppSystem    = "windows"
	AppWallpaper = "windows-wallpaper"
	AppTerminal  = "windows-terminal"
	AppLeopardWM = "leopardwm"
	AppKomorebi  = "komorebi"
	AppWezTerm   = "wezterm"
)

// Paths are the per-user Windows directories the targets write to.
type Paths struct {
	AppData      string // %APPDATA%
	LocalAppData string // %LOCALAPPDATA%
	UserProfile  string // %USERPROFILE%
	// KomorebiConfigHome is %KOMOREBI_CONFIG_HOME%; empty means %USERPROFILE%.
	KomorebiConfigHome string
	// XDGConfigHome is %XDG_CONFIG_HOME%, which Neovim and WezTerm honour on
	// Windows; empty when unset.
	XDGConfigHome string
}

// PathsFromEnv reads Paths from the environment.
func PathsFromEnv() Paths {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(home, "AppData", "Local")
	}
	return Paths{
		AppData:            appData,
		LocalAppData:       local,
		UserProfile:        home,
		KomorebiConfigHome: os.Getenv("KOMOREBI_CONFIG_HOME"),
		XDGConfigHome:      os.Getenv("XDG_CONFIG_HOME"),
	}
}

// Env is everything a target needs from the outside world.
type Env struct {
	Paths Paths
	// System is the registry and desktop. A zero System makes the wallpaper and
	// mode/accent targets fail with winapi.ErrUnsupported.
	System winapi.System
	// StateDir holds the restore record, wallpaper copies and the list of
	// installed files (Aether's data directory).
	StateDir string
	// Run starts an external command and waits for it.
	Run func(name string, args ...string) error
	// LookPath resolves a command name.
	LookPath func(name string) (string, error)
	// Now returns the current time.
	Now func() time.Time
	// Logf receives progress messages.
	Logf func(format string, args ...any)
}

// NativeEnv returns an Env for the current user with the real Windows APIs.
func NativeEnv(stateDir string) *Env {
	system, err := winapi.Native()
	if err != nil {
		system = winapi.System{}
	}
	return &Env{
		Paths:    PathsFromEnv(),
		System:   system,
		StateDir: stateDir,
		Run: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			hideWindow(cmd)
			out, err := cmd.CombinedOutput()
			if err != nil && len(out) > 0 {
				return fmt.Errorf("%s: %w", string(out), err)
			}
			return err
		},
		LookPath: exec.LookPath,
		Now:      time.Now,
		Logf:     log.Printf,
	}
}

func (e *Env) logf(format string, args ...any) {
	if e.Logf != nil {
		e.Logf(format, args...)
	}
}

func (e *Env) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Env) requireSystem() error {
	if e.System.Registry == nil || e.System.Desktop == nil {
		return winapi.ErrUnsupported
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
