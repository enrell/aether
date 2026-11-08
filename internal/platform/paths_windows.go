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
//	%LOCALAPPDATA%\aether\data     restore data and cached downloads
//	%LOCALAPPDATA%\aether\cache    thumbnails, palettes, blurred images
//	%USERPROFILE%\Pictures\Wallpapers  wallpaper library and downloads
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

// downloadDir is the wallpaper library, not the private data directory. A
// downloaded wallpaper has to be usable afterwards: the Local browser scans
// Pictures\Wallpapers, so saving there makes it appear in the gallery and in
// Explorer instead of disappearing under %LOCALAPPDATA%.
func downloadDir() string {
	return wallpaperDir()
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
