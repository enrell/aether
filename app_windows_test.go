//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"aether/internal/testenv"
)

func TestUsableWallpaperFolderRejectsPosixAndRelativePaths(t *testing.T) {
	for input, want := range map[string]bool{
		`C:\Users\enrell\Pictures\Wallpapers`: true,
		`D:\Wallpapers`:                       true,
		`/home/enrell/Wallpapers`:             false,
		`Pictures\Wallpapers`:                 false,
		``:                                    false,
	} {
		if got := usableWallpaperFolder(input); got != want {
			t.Errorf("usableWallpaperFolder(%q) = %t, want %t", input, got, want)
		}
	}
}

func TestWallpaperDownloadDirDefaultsToPictures(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	testenv.WindowsProfile(t, home)

	if got, want := wallpaperDownloadDir(), filepath.Join(home, "Pictures", "Wallpapers"); got != want {
		t.Errorf("wallpaperDownloadDir() = %q, want %q", got, want)
	}
}

func TestWallpaperDownloadDirUsesConfiguredLibrary(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	testenv.WindowsProfile(t, home)

	library := filepath.Join(home, "Pictures", "My Wallpapers")
	configDir := filepath.Join(configHome, "aether")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := fmt.Sprintf(`{"wallpaperFolder":%q}`, library)
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := wallpaperDownloadDir(); got != library {
		t.Errorf("wallpaperDownloadDir() = %q, want %q", got, library)
	}
}
