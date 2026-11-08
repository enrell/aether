//go:build windows

package platform

import (
	"path/filepath"
	"testing"
)

func TestWindowsDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "Local"))
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(key, "")
	}
	for name, tc := range map[string]struct{ got, want string }{
		"config":     {ConfigDir(), filepath.Join(home, "Roaming", "aether")},
		"data":       {DataDir(), filepath.Join(home, "Local", "aether", "data")},
		"cache":      {CacheDir(), filepath.Join(home, "Local", "aether", "cache")},
		"theme":      {ThemeDir(), filepath.Join(home, "Roaming", "aether", "theme")},
		"wallpapers": {WallpaperDir(), filepath.Join(home, "Pictures", "Wallpapers")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", name, tc.got, tc.want)
		}
	}

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if got, want := ConfigDir(), filepath.Join(home, ".config", "aether"); got != want {
		t.Errorf("ConfigDir with XDG_CONFIG_HOME = %q, want %q", got, want)
	}
}
