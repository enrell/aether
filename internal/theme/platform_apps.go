package theme

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"aether/internal/platform"
	"aether/internal/wintheme"
)

// unixOnlyApps have no Windows build or only make sense on a Linux desktop
// (Wayland components, XDG icon themes, Omarchy-specific formats).
var unixOnlyApps = map[string]bool{
	"btop":     true,
	"chromium": true,
	"foot":     true,
	"ghostty":  true,
	"hyprland": true,
	"hyprlock": true,
	"icons":    true,
	"kitty":    true,
	"mako":     true,
	"swayosd":  true,
	"vencord":  true,
	"walker":   true,
	"warp":     true,
	"waybar":   true,
	"wofi":     true,
	"zellij":   true,
}

// windowsOnlyApps are the Windows theming targets.
var windowsOnlyApps = map[string]bool{
	wintheme.AppSystem:    true,
	wintheme.AppWallpaper: true,
	wintheme.AppTerminal:  true,
	wintheme.AppLeopardWM: true,
	wintheme.AppKomorebi:  true,
	wintheme.AppWezTerm:   true,
}

// targetOS is the platform whose apps are offered; tests override it.
var targetOS = runtime.GOOS

// AppSupported reports whether an app template applies on this platform.
// Unsupported apps are hidden from app lists and never rendered.
func AppSupported(app string) bool {
	if targetOS == "windows" {
		return !unixOnlyApps[app]
	}
	return !windowsOnlyApps[app]
}

// SavedTargetSettings returns the targets enabled in the editor (saved in
// settings.json). Apps that were never enabled are excluded.
func SavedTargetSettings() Settings {
	saved, err := platform.ReadJSON[Settings](filepath.Join(platform.ConfigDir(), "settings.json"))
	if err != nil {
		saved = Settings{}
	}
	if saved.IncludedApps == nil {
		saved.IncludedApps = map[string]bool{}
	}
	return saved
}

// newWindowsEnv returns the environment Windows targets run in; tests
// replace it with fakes.
var newWindowsEnv = func() *wintheme.Env {
	return wintheme.NativeEnv(platform.DataDir())
}

// WindowsEnv returns the environment for Windows theming targets.
func WindowsEnv() *wintheme.Env { return newWindowsEnv() }

// applyWindowsTargets hands the rendered Windows templates in themeDir to
// their appliers. A template is only present when its app is included, so
// each target runs only when the user picked it. Failures are collected so
// one broken app config does not stop the others.
func applyWindowsTargets(themeDir, wallpaperPath string) error {
	if targetOS != "windows" {
		return nil
	}
	env := newWindowsEnv()
	var errs []error
	rendered := func(name string) ([]byte, bool) {
		data, err := os.ReadFile(filepath.Join(themeDir, name))
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, err)
			}
			return nil, false
		}
		return data, true
	}
	fail := func(app string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", app, err))
		}
	}

	if data, ok := rendered("windows-wallpaper.json"); ok {
		if wallpaperPath == "" {
			env.Logf("No wallpaper in this theme; leaving the Windows wallpaper alone")
		} else {
			fail(wintheme.AppWallpaper, env.ApplyWallpaper(wallpaperPath, data))
		}
	}
	if data, ok := rendered("windows.json"); ok {
		fail(wintheme.AppSystem, env.ApplySystem(data))
	}
	if data, ok := rendered("windows-terminal.json"); ok {
		_, err := env.ApplyTerminal(data)
		fail(wintheme.AppTerminal, err)
	}
	if data, ok := rendered("leopardwm.toml"); ok {
		_, err := env.ApplyLeopardWM(data)
		fail(wintheme.AppLeopardWM, err)
	}
	if data, ok := rendered("komorebi.json"); ok {
		_, err := env.ApplyKomorebi(data)
		fail(wintheme.AppKomorebi, err)
	}
	for app, file := range map[string]string{
		"alacritty":         "alacritty.toml",
		wintheme.AppWezTerm: "wezterm.lua",
		"neovim":            "neovim.lua",
	} {
		if data, ok := rendered(file); ok {
			_, err := env.Install(app, data)
			fail(app, err)
		}
	}
	return errors.Join(errs...)
}
