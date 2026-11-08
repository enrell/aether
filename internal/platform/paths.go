package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns ~/.config/aether (%APPDATA%\aether on Windows).
func ConfigDir() string {
	return configDir()
}

// CacheDir returns ~/.cache/aether (%LOCALAPPDATA%\aether\cache on Windows).
func CacheDir() string {
	return cacheDir()
}

// DataDir returns the platform-specific data directory for Aether.
// Linux: ~/.local/share/aether (XDG)
// macOS: ~/Library/Application Support/aether/data (via os.UserConfigDir)
// Windows: %LOCALAPPDATA%\aether\data
func DataDir() string {
	return dataDir()
}

// ThemeDir returns ~/.config/aether/theme.
func ThemeDir() string {
	return filepath.Join(ConfigDir(), "theme")
}

// BlueprintDir returns ~/.config/aether/blueprints.
func BlueprintDir() string {
	return filepath.Join(ConfigDir(), "blueprints")
}

// SavedThemesDir returns ~/.config/aether/themes.
func SavedThemesDir() string {
	return filepath.Join(ConfigDir(), "themes")
}

// FavoritesFile returns ~/.config/aether/favorites.json.
func FavoritesFile() string {
	return filepath.Join(ConfigDir(), "favorites.json")
}

// CustomDir returns ~/.config/aether/custom.
func CustomDir() string {
	return filepath.Join(ConfigDir(), "custom")
}

// OmarchyThemesDir returns Omarchy's native ~/.config/omarchy/themes path.
func OmarchyThemesDir() string {
	return filepath.Join(homeDir(), ".config", "omarchy", "themes")
}

// OmarchyThemeDir returns ~/.config/omarchy/themes/aether.
func OmarchyThemeDir() string {
	return filepath.Join(OmarchyThemesDir(), "aether")
}

// WallpaperDir returns ~/Wallpapers, the default local scan directory
// (%USERPROFILE%\Pictures\Wallpapers on Windows).
func WallpaperDir() string {
	return wallpaperDir()
}

// DownloadDir returns the directory downloaded wallpapers are saved to.
// Linux: ~/.local/share/aether/wallpapers
// Windows: %USERPROFILE%\Pictures\Wallpapers, the same folder the Local browser
// scans, so a download is visible in the library and in Explorer.
func DownloadDir() string {
	return downloadDir()
}

// ThumbnailDir returns ~/.cache/aether/thumbnails.
func ThumbnailDir() string {
	return filepath.Join(CacheDir(), "thumbnails")
}

// ColorCacheDir returns ~/.cache/aether/color-cache.
func ColorCacheDir() string {
	return filepath.Join(CacheDir(), "color-cache")
}

// BlurDir returns ~/.cache/aether/blur.
func BlurDir() string {
	return filepath.Join(CacheDir(), "blur")
}

// EnsureAllDirs creates all directories required by Aether. It does not create
// WallpaperDir because that is user-managed, except on Windows where
// DownloadDir is the wallpaper library and downloads need it to exist.
func EnsureAllDirs() error {
	dirs := []string{
		ConfigDir(),
		CacheDir(),
		DataDir(),
		ThemeDir(),
		BlueprintDir(),
		SavedThemesDir(),
		CustomDir(),
		DownloadDir(),
		ThumbnailDir(),
		ColorCacheDir(),
		BlurDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// HomeDir returns the current user's home directory.
func HomeDir() string {
	return homeDir()
}

// homeDir returns the current user's home directory.
func homeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			// os.UserHomeDir reads %USERPROFILE%; keep the Windows shape if
			// that lookup failed instead of producing /home/<user>.
			if profile := os.Getenv("USERPROFILE"); profile != "" {
				return profile
			}
			return filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
		}
		if runtime.GOOS == "darwin" {
			return filepath.Join("/Users", os.Getenv("USER"))
		}
		return filepath.Join("/home", os.Getenv("USER"))
	}
	return dir
}
