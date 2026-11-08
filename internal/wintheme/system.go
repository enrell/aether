package wintheme

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"aether/internal/color"
	"aether/internal/platform"
	"aether/internal/winapi"

	"github.com/tailscale/hujson"
)

// Registry locations below HKEY_CURRENT_USER.
const (
	personalizeKey   = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	accentKey        = `Software\Microsoft\Windows\CurrentVersion\Explorer\Accent`
	dwmKey           = `Software\Microsoft\Windows\DWM`
	desktopKey       = `Control Panel\Desktop`
	desktopColorsKey = `Control Panel\Colors`
)

// SystemConfig is the rendered windows.json template.
type SystemConfig struct {
	// Mode is "light" or "dark" for both apps and the shell.
	Mode string `json:"mode"`
	// AppsMode and SystemMode override Mode for one side.
	AppsMode   string `json:"appsMode,omitempty"`
	SystemMode string `json:"systemMode,omitempty"`
	// Accent is the accent colour (#rrggbb). Empty leaves the accent alone.
	Accent string `json:"accent"`
	// AccentOnTaskbar and AccentOnTitleBars show the accent on Start and the
	// taskbar, and on title bars and window borders. Unset leaves the
	// current choice.
	AccentOnTaskbar   *bool `json:"accentOnTaskbar,omitempty"`
	AccentOnTitleBars *bool `json:"accentOnTitleBars,omitempty"`
}

func decodeTemplate(rendered []byte, v any) error {
	std, err := hujson.Standardize(append([]byte(nil), rendered...))
	if err != nil {
		return err
	}
	return json.Unmarshal(std, v)
}

func lightModeValue(mode string) (uint32, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "light":
		return 1, nil
	case "dark":
		return 0, nil
	}
	return 0, fmt.Errorf("mode must be \"light\" or \"dark\", got %q", mode)
}

// ApplySystem sets the light/dark mode and the accent colour, then tells
// running applications that the colour set changed.
func (e *Env) ApplySystem(rendered []byte) error {
	var cfg SystemConfig
	if err := decodeTemplate(rendered, &cfg); err != nil {
		return fmt.Errorf("read windows.json template: %w", err)
	}
	if err := e.requireSystem(); err != nil {
		return err
	}
	record, err := e.loadRestore()
	if err != nil {
		return err
	}

	type write struct {
		key, name string
		value     winapi.Value
	}
	var writes []write
	for _, side := range []struct{ name, mode string }{
		{"AppsUseLightTheme", firstNonEmpty(cfg.AppsMode, cfg.Mode)},
		{"SystemUsesLightTheme", firstNonEmpty(cfg.SystemMode, cfg.Mode)},
	} {
		if side.mode == "" {
			continue
		}
		v, err := lightModeValue(side.mode)
		if err != nil {
			return err
		}
		writes = append(writes, write{personalizeKey, side.name, winapi.DWORD(v)})
	}

	if cfg.Accent != "" {
		if !color.IsHexColor(cfg.Accent) {
			return fmt.Errorf("accent %q is not a #rrggbb colour", cfg.Accent)
		}
		palette := AccentPalette(cfg.Accent)
		base, dark1 := palette[3], palette[4]
		writes = append(writes,
			write{accentKey, "AccentPalette", winapi.Binary(accentPaletteBytes(palette))},
			write{accentKey, "AccentColorMenu", winapi.DWORD(abgr(base))},
			write{accentKey, "StartColorMenu", winapi.DWORD(abgr(dark1))},
			write{dwmKey, "AccentColor", winapi.DWORD(abgr(base))},
			write{dwmKey, "ColorizationColor", winapi.DWORD(argb(base, 0xc4))},
			write{dwmKey, "ColorizationAfterglow", winapi.DWORD(argb(base, 0xc4))},
		)
		// "Automatically pick an accent colour from my background" would
		// replace the accent on the next wallpaper change.
		if v, ok, err := e.System.Registry.Get(desktopKey, "AutoColorization"); err == nil && ok && v.Kind == winapi.KindDWORD && v.DWORD != 0 {
			writes = append(writes, write{desktopKey, "AutoColorization", winapi.DWORD(0)})
		}
	}
	if cfg.AccentOnTaskbar != nil {
		writes = append(writes, write{personalizeKey, "ColorPrevalence", winapi.DWORD(boolDWORD(*cfg.AccentOnTaskbar))})
	}
	if cfg.AccentOnTitleBars != nil {
		writes = append(writes, write{dwmKey, "ColorPrevalence", winapi.DWORD(boolDWORD(*cfg.AccentOnTitleBars))})
	}

	changed := false
	for _, w := range writes {
		did, err := e.setRegistry(record, w.key, w.name, w.value)
		if err != nil {
			return err
		}
		changed = changed || did
	}
	if !changed {
		e.logf("Windows mode and accent are already up to date")
		return nil
	}
	if err := e.System.Desktop.BroadcastSettingChange("ImmersiveColorSet"); err != nil {
		return err
	}
	e.logf("Applied Windows %s mode and accent %s", cfg.Mode, cfg.Accent)
	return nil
}

// WallpaperConfig is the rendered windows-wallpaper.json template.
type WallpaperConfig struct {
	// Style is fill, fit, stretch, center, tile or span.
	Style string `json:"style"`
	// DesktopColor is the solid colour around a fitted or centred wallpaper.
	// Empty leaves it alone.
	DesktopColor string `json:"desktopColor"`
}

var wallpaperStyles = map[string][2]string{
	// style: WallpaperStyle, TileWallpaper
	"fill":    {"10", "0"},
	"fit":     {"6", "0"},
	"stretch": {"2", "0"},
	"center":  {"0", "0"},
	"tile":    {"0", "1"},
	"span":    {"22", "0"},
}

// ApplyWallpaper sets path as the wallpaper on every monitor.
func (e *Env) ApplyWallpaper(path string, rendered []byte) error {
	var cfg WallpaperConfig
	if len(rendered) > 0 {
		if err := decodeTemplate(rendered, &cfg); err != nil {
			return fmt.Errorf("read windows-wallpaper.json template: %w", err)
		}
	}
	if err := e.requireSystem(); err != nil {
		return err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("wallpaper %s is not a readable file", abs)
	}
	record, err := e.loadRestore()
	if err != nil {
		return err
	}
	if record.Wallpaper == nil {
		if err := e.recordWallpaper(record); err != nil {
			return err
		}
	}

	if cfg.Style != "" {
		style, ok := wallpaperStyles[strings.ToLower(cfg.Style)]
		if !ok {
			return fmt.Errorf("unknown wallpaper style %q (use fill, fit, stretch, center, tile or span)", cfg.Style)
		}
		if _, err := e.setRegistry(record, desktopKey, "WallpaperStyle", winapi.String(style[0])); err != nil {
			return err
		}
		if _, err := e.setRegistry(record, desktopKey, "TileWallpaper", winapi.String(style[1])); err != nil {
			return err
		}
	}
	if cfg.DesktopColor != "" {
		if !color.IsHexColor(cfg.DesktopColor) {
			return fmt.Errorf("desktopColor %q is not a #rrggbb colour", cfg.DesktopColor)
		}
		rgb := color.HexToRGB(cfg.DesktopColor)
		r, g, b := uint8(rgb.R), uint8(rgb.G), uint8(rgb.B)
		if _, err := e.setRegistry(record, desktopColorsKey, "Background", winapi.String(fmt.Sprintf("%d %d %d", r, g, b))); err != nil {
			return err
		}
		if err := e.System.Desktop.SetDesktopColor(r, g, b); err != nil {
			return err
		}
	}
	if err := e.System.Desktop.SetWallpaper(abs); err != nil {
		return err
	}
	e.logf("Set Windows wallpaper to %s", abs)
	return nil
}

// recordWallpaper stores the current wallpaper, with a private copy of the
// image, before Aether replaces it.
func (e *Env) recordWallpaper(record *restoreRecord) error {
	current, err := e.System.Desktop.Wallpaper()
	if err != nil {
		return err
	}
	entry := &wallpaperEntry{Path: current}
	if current != "" && exists(current) {
		dir := filepath.Join(e.StateDir, "windows-restore")
		ext := filepath.Ext(current)
		if ext == "" || isWindowsTranscoded(current) {
			ext = ".jpg"
		}
		copyPath := filepath.Join(dir, "wallpaper"+ext)
		if err := platform.CopyFile(current, copyPath); err != nil {
			e.logf("Warning: could not keep a copy of the current wallpaper: %v", err)
		} else {
			entry.Copy = copyPath
		}
	}
	record.Wallpaper = entry
	return e.saveRestore(record)
}

// AccentPalette derives the eight AccentPalette entries Windows stores for an
// accent: three lighter shades, the accent, three darker shades and a fixed
// complementary orange. Shades step OKLCH lightness like the built-in
// palettes do.
func AccentPalette(accent string) [8]string {
	lch := color.HexToOKLCH(accent)
	shade := func(dl float64) string {
		c := lch
		c.L = math.Max(0.12, math.Min(0.95, c.L+dl))
		// Very light and very dark shades carry less chroma.
		edge := math.Abs(dl) / 0.4
		c.C = lch.C * (1 - 0.35*math.Min(1, edge))
		return color.OKLCHToHex(c)
	}
	return [8]string{
		shade(+0.30), shade(+0.19), shade(+0.08),
		accent,
		shade(-0.06), shade(-0.17), shade(-0.28),
		"#f7630c",
	}
}

func accentPaletteBytes(palette [8]string) []byte {
	out := make([]byte, 0, 32)
	for _, hex := range palette {
		rgb := color.HexToRGB(hex)
		out = append(out, uint8(rgb.R), uint8(rgb.G), uint8(rgb.B), 0)
	}
	return out
}

// abgr packs a colour as 0xAABBGGRR, the layout of AccentColorMenu.
func abgr(hex string) uint32 {
	rgb := color.HexToRGB(hex)
	return 0xff000000 | uint32(rgb.B)<<16 | uint32(rgb.G)<<8 | uint32(rgb.R)
}

// argb packs a colour as 0xAARRGGBB, the layout of ColorizationColor.
func argb(hex string, alpha uint8) uint32 {
	rgb := color.HexToRGB(hex)
	return uint32(alpha)<<24 | uint32(rgb.R)<<16 | uint32(rgb.G)<<8 | uint32(rgb.B)
}

func boolDWORD(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
