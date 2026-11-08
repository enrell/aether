//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(homeDir(), ".config")
	}
	return filepath.Join(dir, "aether")
}

func cacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = filepath.Join(homeDir(), ".cache")
	}
	return filepath.Join(dir, "aether")
}

func dataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "aether")
	}
	if runtime.GOOS == "darwin" {
		// macOS: use ~/Library/Application Support
		dir, err := os.UserConfigDir()
		if err == nil {
			return filepath.Join(dir, "aether", "data")
		}
	}
	return filepath.Join(homeDir(), ".local", "share", "aether")
}

func wallpaperDir() string {
	return filepath.Join(homeDir(), "Wallpapers")
}
