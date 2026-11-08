package theme

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aether/internal/platform"
	"aether/internal/template"
	"aether/internal/winapi"
	"aether/internal/wintheme"
)

func withTargetOS(t *testing.T, goos string) {
	t.Helper()
	previous := targetOS
	targetOS = goos
	t.Cleanup(func() { targetOS = previous })
}

func TestAppSupportedPerPlatform(t *testing.T) {
	withTargetOS(t, "linux")
	for _, app := range []string{"hyprland", "kitty", "waybar", "icons", "alacritty", "neovim", "colors"} {
		if !AppSupported(app) {
			t.Errorf("linux: %s should be supported", app)
		}
	}
	for _, app := range []string{"windows", "windows-terminal", "leopardwm", "komorebi", "wezterm", "windows-wallpaper"} {
		if AppSupported(app) {
			t.Errorf("linux: %s should be hidden", app)
		}
	}

	withTargetOS(t, "windows")
	for _, app := range []string{"windows", "windows-terminal", "leopardwm", "komorebi", "wezterm", "alacritty", "neovim", "vscode", "zed", "colors"} {
		if !AppSupported(app) {
			t.Errorf("windows: %s should be supported", app)
		}
	}
	for _, app := range []string{"hyprland", "hyprlock", "waybar", "kitty", "ghostty", "foot", "mako", "icons", "btop", "zellij"} {
		if AppSupported(app) {
			t.Errorf("windows: %s should be hidden", app)
		}
	}
}

// Every Windows app in templates/ must be known to the app-name map and the
// platform filter, or it would leak into the Linux app list.
func TestWindowsTemplatesAreRegistered(t *testing.T) {
	withTargetOS(t, "linux")
	entries, err := os.ReadDir(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		app := GetAppNameFromFileName(e.Name())
		if windowsOnlyApps[app] {
			found[app] = true
			if _, ok := templateAppNameMap[e.Name()]; !ok {
				t.Errorf("%s is missing from templateAppNameMap", e.Name())
			}
		}
	}
	for app := range windowsOnlyApps {
		if !found[app] {
			t.Errorf("no template for Windows app %s", app)
		}
	}
}

type windowsFixture struct {
	env      *wintheme.Env
	reg      *winapi.MemoryRegistry
	desk     *winapi.FakeDesktop
	runs     [][]string
	terminal string
	leopard  string
}

func newWindowsFixture(t *testing.T) *windowsFixture {
	t.Helper()
	root := t.TempDir()
	paths := wintheme.Paths{
		AppData:      filepath.Join(root, "Roaming"),
		LocalAppData: filepath.Join(root, "Local"),
		UserProfile:  root,
	}
	system, reg, desk := winapi.FakeSystem()
	f := &windowsFixture{reg: reg, desk: desk}
	f.env = &wintheme.Env{
		Paths:    paths,
		System:   system,
		StateDir: filepath.Join(root, "Local", "aether", "data"),
		Run: func(name string, args ...string) error {
			f.runs = append(f.runs, append([]string{name}, args...))
			return nil
		},
		LookPath: func(name string) (string, error) {
			if name == "lwm" {
				return "lwm.exe", nil
			}
			return "", errors.New("not found")
		},
		Now:  time.Now,
		Logf: t.Logf,
	}
	f.terminal = filepath.Join(paths.LocalAppData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
	f.leopard = filepath.Join(paths.AppData, "leopardwm", "config", "config.toml")
	mustWrite(t, f.terminal, `{"defaultProfile": "{x}", "profiles": {"defaults": {}, "list": [{"guid": "{x}", "name": "PowerShell"}]}, "schemes": []}`)
	mustWrite(t, f.leopard, "[layout]\ngap = 10\n\n[appearance]\nlanguage = \"en\"\n\n[hotkeys]\n\"Win+H\" = \"focus_left\"\n")
	for _, dir := range []string{filepath.Join(paths.AppData, "alacritty"), filepath.Join(paths.LocalAppData, "nvim"), filepath.Join(root, ".config", "wezterm")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	previous := newWindowsEnv
	newWindowsEnv = func() *wintheme.Env { return f.env }
	t.Cleanup(func() { newWindowsEnv = previous })
	return f
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func renderRealTemplate(t *testing.T, dir, name string, vars map[string]string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "templates", name))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, name), template.ProcessTemplate(string(content), vars))
}

func testVariables(lightMode bool) map[string]string {
	roles := template.ColorRoles{
		Background: "#1a1b26", Foreground: "#c0caf5",
		Black: "#15161e", Red: "#f7768e", Green: "#9ece6a", Yellow: "#e0af68",
		Blue: "#7aa2f7", Magenta: "#bb9af7", Cyan: "#7dcfff", White: "#a9b1d6",
		BrightBlack: "#414868", BrightRed: "#ff899d", BrightGreen: "#9fe044", BrightYellow: "#faba4a",
		BrightBlue: "#8db0ff", BrightMagenta: "#c7a9ff", BrightCyan: "#a4daff", BrightWhite: "#c0caf5",
		Accent: "#7aa2f7",
	}
	return template.BuildVariables(roles, lightMode, nil)
}

func TestApplyWindowsTargetsWithShippedTemplates(t *testing.T) {
	withTargetOS(t, "windows")
	f := newWindowsFixture(t)
	themeDir := t.TempDir()
	vars := testVariables(false)
	komorebi := filepath.Join(f.env.Paths.UserProfile, "komorebi.json")
	mustWrite(t, komorebi, `{"border": true, "theme": {"palette": "Base16", "name": "Ashes", "unfocused_border": "Base03"}}`)
	for _, name := range []string{"windows.json", "windows-wallpaper.json", "windows-terminal.json", "leopardwm.toml", "komorebi.json", "alacritty.toml", "wezterm.lua", "neovim.lua"} {
		renderRealTemplate(t, themeDir, name, vars)
	}
	wallpaper := filepath.Join(themeDir, "backgrounds", "wall.png")
	mustWrite(t, wallpaper, "png")

	if err := applyWindowsTargets(themeDir, wallpaper); err != nil {
		t.Fatal(err)
	}

	if f.desk.WallpaperSet != wallpaper {
		t.Errorf("wallpaper = %q", f.desk.WallpaperSet)
	}
	if v, ok, _ := f.reg.Get(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme"); !ok || v.DWORD != 0 {
		t.Errorf("AppsUseLightTheme = %+v %v", v, ok)
	}
	if v, _, _ := f.reg.Get(`Software\Microsoft\Windows\DWM`, "AccentColor"); v.DWORD != 0xfff7a27a {
		t.Errorf("AccentColor = %#x, want ABGR of #7aa2f7", v.DWORD)
	}

	var settings struct {
		Theme    string `json:"theme"`
		Profiles struct {
			Defaults map[string]any `json:"defaults"`
		} `json:"profiles"`
		Schemes []map[string]string `json:"schemes"`
		Themes  []map[string]any    `json:"themes"`
	}
	data, _ := os.ReadFile(f.terminal)
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("settings.json: %v\n%s", err, data)
	}
	if len(settings.Schemes) != 1 || settings.Schemes[0]["purple"] != "#bb9af7" || settings.Profiles.Defaults["colorScheme"] != "Aether" {
		t.Errorf("settings.json = %s", data)
	}
	if settings.Theme != "Aether" || len(settings.Themes) != 1 {
		t.Errorf("window theme not installed: %s", data)
	}

	leopard, _ := os.ReadFile(f.leopard)
	for _, want := range []string{`active_border_color = "7AA2F7"`, `tab_strip_bg = "1A1B26"`, `"Win+H" = "focus_left"`} {
		if !strings.Contains(string(leopard), want) {
			t.Errorf("config.toml lacks %s:\n%s", want, leopard)
		}
	}
	if len(f.runs) != 1 || f.runs[0][0] != "lwm.exe" || f.runs[0][1] != "reload" {
		t.Errorf("runs = %v", f.runs)
	}

	var kconf struct {
		Border bool `json:"border"`
		Theme  struct {
			Palette         string            `json:"palette"`
			Name            *string           `json:"name"`
			UnfocusedBorder string            `json:"unfocused_border"`
			Colours         map[string]string `json:"colours"`
		} `json:"theme"`
	}
	data, _ = os.ReadFile(komorebi)
	if err := json.Unmarshal(data, &kconf); err != nil {
		t.Fatalf("komorebi.json: %v", err)
	}
	if !kconf.Border || kconf.Theme.Palette != "Custom" || kconf.Theme.Name != nil || kconf.Theme.UnfocusedBorder != "Base03" || len(kconf.Theme.Colours) != 16 || kconf.Theme.Colours["base_0d"] != "#7aa2f7" {
		t.Errorf("komorebi.json = %s", data)
	}

	for _, path := range []string{
		filepath.Join(f.env.Paths.AppData, "alacritty", "aether.toml"),
		filepath.Join(f.env.Paths.UserProfile, ".config", "wezterm", "aether.lua"),
		filepath.Join(f.env.Paths.LocalAppData, "nvim", "lua", "plugins", "aether.lua"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("not installed: %v", err)
			continue
		}
		if strings.Contains(string(content), "{background}") {
			t.Errorf("%s contains unrendered variables", path)
		}
	}
}

func TestApplyWindowsTargetsLightMode(t *testing.T) {
	withTargetOS(t, "windows")
	f := newWindowsFixture(t)
	themeDir := t.TempDir()
	renderRealTemplate(t, themeDir, "windows.json", testVariables(true))
	if err := applyWindowsTargets(themeDir, ""); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := f.reg.Get(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "SystemUsesLightTheme"); v.DWORD != 1 {
		t.Errorf("SystemUsesLightTheme = %d, want 1", v.DWORD)
	}
	if f.desk.WallpaperSet != "" || f.reg.Writes == 0 {
		t.Errorf("only mode and accent should change: wallpaper %q, writes %d", f.desk.WallpaperSet, f.reg.Writes)
	}
}

func TestApplyWindowsTargetsIsNoOpElsewhere(t *testing.T) {
	withTargetOS(t, "linux")
	called := false
	previous := newWindowsEnv
	newWindowsEnv = func() *wintheme.Env { called = true; return nil }
	t.Cleanup(func() { newWindowsEnv = previous })
	if err := applyWindowsTargets(t.TempDir(), ""); err != nil || called {
		t.Errorf("applyWindowsTargets on linux: err=%v called=%v", err, called)
	}
}

func TestProcessTemplatesSkipsUnsupportedApps(t *testing.T) {
	setupWriterTestEnv(t)
	withTargetOS(t, "windows")
	out := t.TempDir()
	stale := filepath.Join(out, "kitty.conf")
	mustWrite(t, stale, "old")
	writer := NewWriter(omarchyV4TestTemplates, "testdata/v4")
	vars := testVariables(false)
	if err := writer.processTemplates(vars, out, Settings{IncludedApps: map[string]bool{"kitty": true, "icons": true}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("kitty.conf rendered or kept on Windows")
	}
	if _, err := os.Stat(filepath.Join(out, "icons.theme")); !os.IsNotExist(err) {
		t.Error("icons.theme rendered on Windows")
	}
}

func TestSavedTargetSettings(t *testing.T) {
	setupWriterTestEnv(t)
	if got := SavedTargetSettings(); got.IncludedApps == nil || len(got.IncludedApps) != 0 || got.includesApp("windows") {
		t.Errorf("without settings.json: %+v", got)
	}
	mustWrite(t, filepath.Join(platform.ConfigDir(), "settings.json"), `{"includedApps": {"windows-terminal": true}, "wallpaperFolder": "x"}`)
	got := SavedTargetSettings()
	if !got.includesApp("windows-terminal") || got.includesApp("windows") || got.includesApp("windows-wallpaper") {
		t.Errorf("saved targets = %+v", got.IncludedApps)
	}
}
