//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

// Windows keeps settings in the roaming profile and regeneratable or large
// files in the local profile:
//
//	%APPDATA%\aether               settings, blueprints, generated theme
//	%LOCALAPPDATA%\aether\data     downloaded wallpapers and restore data
//	%LOCALAPPDATA%\aether\cache    thumbnails, palettes, blurred images
//
// XDG_CONFIG_HOME, XDG_DATA_HOME and XDG_CACHE_HOME win when set, as they do
// for Neovim and WezTerm on Windows, so a dotfiles setup can keep Aether in
// ~/.config like on Linux. Environment variables are read first so tests and
// portable setups can redirect every path.

func configDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "aether")
	}
	return filepath.Join(RoamingAppData(), "aether")
}

func cacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "aether")
	}
	return filepath.Join(LocalAppData(), "aether", "cache")
}

func dataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "aether")
	}
	return filepath.Join(LocalAppData(), "aether", "data")
}

func wallpaperDir() string {
	return filepath.Join(homeDir(), "Pictures", "Wallpapers")
}

// RoamingAppData returns %APPDATA%.
func RoamingAppData() string {
	if dir := os.Getenv("APPDATA"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return dir
	}
	return filepath.Join(homeDir(), "AppData", "Roaming")
}

// LocalAppData returns %LOCALAPPDATA%.
func LocalAppData() string {
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return dir
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return dir
	}
	return filepath.Join(homeDir(), "AppData", "Local")
}
