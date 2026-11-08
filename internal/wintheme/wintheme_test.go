package wintheme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"aether/internal/winapi"
)

// testEnv returns an Env whose Windows profile lives in a temp directory and
// whose registry and desktop are fakes.
func testEnv(t *testing.T) (*Env, *winapi.MemoryRegistry, *winapi.FakeDesktop, *[][]string) {
	t.Helper()
	root := t.TempDir()
	paths := Paths{
		AppData:      filepath.Join(root, "AppData", "Roaming"),
		LocalAppData: filepath.Join(root, "AppData", "Local"),
		UserProfile:  root,
	}
	system, reg, desk := winapi.FakeSystem()
	var runs [][]string
	env := &Env{
		Paths:    paths,
		System:   system,
		StateDir: filepath.Join(paths.LocalAppData, "aether", "data"),
		Run: func(name string, args ...string) error {
			runs = append(runs, append([]string{name}, args...))
			return nil
		},
		LookPath: func(name string) (string, error) { return "", errors.New("not found") },
		Now:      func() time.Time { return time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC) },
		Logf:     t.Logf,
	}
	return env, reg, desk, &runs
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// seedWindowsDefaults fills the registry with a stock Windows 11 setup.
func seedWindowsDefaults(t *testing.T, reg *winapi.MemoryRegistry, desk *winapi.FakeDesktop, wallpaper string) {
	t.Helper()
	sets := []struct {
		key, name string
		v         winapi.Value
	}{
		{personalizeKey, "AppsUseLightTheme", winapi.DWORD(1)},
		{personalizeKey, "SystemUsesLightTheme", winapi.DWORD(0)},
		{personalizeKey, "ColorPrevalence", winapi.DWORD(0)},
		{accentKey, "AccentPalette", winapi.Binary([]byte{0x99, 0xeb, 0xff, 0, 0x4c, 0xc2, 0xff, 0, 0, 0x91, 0xf8, 0, 0, 0x78, 0xd4, 0, 0, 0x67, 0xc0, 0, 0, 0x3e, 0x92, 0, 0, 0x1a, 0x68, 0, 0xf7, 0x63, 0x0c, 0})},
		{accentKey, "AccentColorMenu", winapi.DWORD(0xffd47800)},
		{accentKey, "StartColorMenu", winapi.DWORD(0xffc06700)},
		{dwmKey, "AccentColor", winapi.DWORD(0xffd47800)},
		{dwmKey, "ColorizationColor", winapi.DWORD(0xc40078d4)},
		{dwmKey, "ColorizationAfterglow", winapi.DWORD(0xc40078d4)},
		{desktopKey, "WallpaperStyle", winapi.String("10")},
		{desktopKey, "TileWallpaper", winapi.String("0")},
		{desktopColorsKey, "Background", winapi.String("0 0 0")},
	}
	for _, s := range sets {
		if err := reg.Set(s.key, s.name, s.v); err != nil {
			t.Fatal(err)
		}
	}
	reg.Writes = 0
	desk.WallpaperSet = wallpaper
}

func registryDump(t *testing.T, reg *winapi.MemoryRegistry) map[string]winapi.Value {
	t.Helper()
	out := map[string]winapi.Value{}
	for _, key := range []string{personalizeKey, accentKey, dwmKey, desktopKey, desktopColorsKey} {
		for _, name := range []string{"AppsUseLightTheme", "SystemUsesLightTheme", "ColorPrevalence", "AccentPalette", "AccentColorMenu", "StartColorMenu", "AccentColor", "ColorizationColor", "ColorizationAfterglow", "WallpaperStyle", "TileWallpaper", "Background", "AutoColorization"} {
			if v, ok, _ := reg.Get(key, name); ok {
				out[key+`\`+name] = v
			}
		}
	}
	return out
}

func TestApplySystemWritesModeAndAccent(t *testing.T) {
	env, reg, desk, _ := testEnv(t)
	seedWindowsDefaults(t, reg, desk, "")

	if err := env.ApplySystem([]byte(`{"mode": "dark", "accent": "#e06c75", "accentOnTitleBars": true}`)); err != nil {
		t.Fatal(err)
	}
	get := func(key, name string) winapi.Value {
		v, ok, _ := reg.Get(key, name)
		if !ok {
			t.Fatalf(`%s\%s missing`, key, name)
		}
		return v
	}
	if v := get(personalizeKey, "AppsUseLightTheme"); v.DWORD != 0 {
		t.Errorf("AppsUseLightTheme = %d, want 0", v.DWORD)
	}
	if v := get(accentKey, "AccentColorMenu"); v.DWORD != 0xff756ce0 {
		t.Errorf("AccentColorMenu = %#x, want ABGR 0xff756ce0", v.DWORD)
	}
	if v := get(dwmKey, "ColorizationColor"); v.DWORD != 0xc4e06c75 {
		t.Errorf("ColorizationColor = %#x, want ARGB 0xc4e06c75", v.DWORD)
	}
	if v := get(dwmKey, "ColorPrevalence"); v.DWORD != 1 {
		t.Errorf("DWM ColorPrevalence = %d, want 1", v.DWORD)
	}
	if v := get(accentKey, "AccentPalette"); len(v.Binary) != 32 || v.Binary[12] != 0xe0 || v.Binary[13] != 0x6c || v.Binary[14] != 0x75 {
		t.Errorf("AccentPalette = % x", v.Binary)
	}
	if _, ok, _ := reg.Get(desktopKey, "AutoColorization"); ok {
		t.Error("AutoColorization was created although it was absent")
	}
	if !reflect.DeepEqual(desk.Broadcasts, []string{"ImmersiveColorSet"}) {
		t.Errorf("broadcasts = %v", desk.Broadcasts)
	}

	// Applying the same theme again changes nothing.
	writes := reg.Writes
	if err := env.ApplySystem([]byte(`{"mode": "dark", "accent": "#e06c75", "accentOnTitleBars": true}`)); err != nil {
		t.Fatal(err)
	}
	if reg.Writes != writes || len(desk.Broadcasts) != 1 {
		t.Errorf("second apply wrote %d values and broadcast %d times", reg.Writes-writes, len(desk.Broadcasts))
	}
}

func TestApplySystemDisablesAutomaticAccent(t *testing.T) {
	env, reg, desk, _ := testEnv(t)
	seedWindowsDefaults(t, reg, desk, "")
	_ = reg.Set(desktopKey, "AutoColorization", winapi.DWORD(1))
	if err := env.ApplySystem([]byte(`{"mode": "light", "accent": "#336699"}`)); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := reg.Get(desktopKey, "AutoColorization"); v.DWORD != 0 {
		t.Errorf("AutoColorization = %d, want 0", v.DWORD)
	}
	if v, _, _ := reg.Get(personalizeKey, "SystemUsesLightTheme"); v.DWORD != 1 {
		t.Errorf("SystemUsesLightTheme = %d, want 1", v.DWORD)
	}
}

func TestApplySystemRejectsBadInput(t *testing.T) {
	env, _, _, _ := testEnv(t)
	for _, tpl := range []string{`{"mode": "dim"}`, `{"mode": "dark", "accent": "blue"}`, `{`} {
		if err := env.ApplySystem([]byte(tpl)); err == nil {
			t.Errorf("ApplySystem(%s) succeeded", tpl)
		}
	}
	env.System = winapi.System{}
	if err := env.ApplySystem([]byte(`{"mode": "dark"}`)); !errors.Is(err, winapi.ErrUnsupported) {
		t.Errorf("without a system: %v", err)
	}
}

func TestAccentPaletteMatchesWindowsShape(t *testing.T) {
	p := AccentPalette("#0078d4")
	if p[3] != "#0078d4" || p[7] != "#f7630c" {
		t.Errorf("palette = %v", p)
	}
	prev := 256.0
	for i := 0; i < 7; i++ {
		l := luminance(p[i])
		if l >= prev {
			t.Errorf("shade %d (%s) is not darker than shade %d", i, p[i], i-1)
		}
		prev = l
	}
}

func luminance(hex string) float64 {
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
}

func TestApplyWallpaper(t *testing.T) {
	env, reg, desk, _ := testEnv(t)
	original := filepath.Join(env.Paths.UserProfile, "old.jpg")
	writeFile(t, original, "old image")
	seedWindowsDefaults(t, reg, desk, original)
	wall := filepath.Join(env.Paths.AppData, "aether", "theme", "backgrounds", "new.png")
	writeFile(t, wall, "new image")

	if err := env.ApplyWallpaper(wall, []byte(`{"style": "fit", "desktopColor": "#102030"}`)); err != nil {
		t.Fatal(err)
	}
	if desk.WallpaperSet != wall {
		t.Errorf("wallpaper = %q", desk.WallpaperSet)
	}
	if v, _, _ := reg.Get(desktopKey, "WallpaperStyle"); v.String != "6" {
		t.Errorf("WallpaperStyle = %q, want 6 (fit)", v.String)
	}
	if v, _, _ := reg.Get(desktopColorsKey, "Background"); v.String != "16 32 48" {
		t.Errorf("Background = %q", v.String)
	}
	if desk.Color == nil || *desk.Color != [3]uint8{16, 32, 48} {
		t.Errorf("desktop colour = %v", desk.Color)
	}
	record, _ := env.loadRestore()
	if record.Wallpaper == nil || record.Wallpaper.Path != original || readFile(t, record.Wallpaper.Copy) != "old image" {
		t.Errorf("restore record = %+v", record.Wallpaper)
	}
	if err := env.ApplyWallpaper(wall, []byte(`{"style": "zoom"}`)); err == nil {
		t.Error("unknown style accepted")
	}
	if err := env.ApplyWallpaper(filepath.Join(env.Paths.UserProfile, "missing.jpg"), nil); err == nil {
		t.Error("missing wallpaper accepted")
	}
}

// terminalFixture mirrors a stock settings.json; <SP> marks the trailing
// spaces Windows Terminal writes after some keys.
var terminalFixture = strings.ReplaceAll(`{
    "$help": "https://aka.ms/terminal-documentation",
    "$schema": "https://aka.ms/terminal-profiles-schema",
    "actions": [],
    "copyOnSelect": false,
    "defaultProfile": "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
    "keybindings":<SP>
    [
        {
            "id": "Terminal.CopyToClipboard",
            "keys": "ctrl+c"
        }
    ],
    "profiles":<SP>
    {
        "defaults": {},
        "list":<SP>
        [
            {
                "commandline": "wsl -d archlinux",
                "guid": "{2ef7bc85-e0bc-436a-a742-5aeaeabe58b2}",
                "name": "Archlinux"
            },
            {
                "guid": "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
                "hidden": false,
                "name": "PowerShell",
                "source": "Windows.Terminal.PowershellCore"
            }
        ]
    },
    "schemes": [],
    "themes": []
}
`, "<SP>", " ")

const terminalTemplate = `// comment
{
  "applyTo": "defaults",
  "scheme": {"name": "Aether", "background": "#101010", "foreground": "#e0e0e0", "black": "#000000"},
  "theme": {"name": "Aether", "window": {"applicationTheme": "dark"}, "tabRow": {"background": "#0c0c0c"}}
}`

func terminalFile(env *Env) string {
	return filepath.Join(env.Paths.LocalAppData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")
}

func decodeJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, s)
	}
	return v
}

func TestApplyTerminal(t *testing.T) {
	env, _, _, _ := testEnv(t)
	store := terminalFile(env)
	unpackaged := filepath.Join(env.Paths.LocalAppData, "Microsoft", "Windows Terminal", "settings.json")
	writeFile(t, store, terminalFixture)
	writeFile(t, unpackaged, `{"profiles": {"list": []}, "schemes": [{"name": "Aether", "background": "#ffffff"}, {"name": "Mine"}]}`)

	changed, err := env.ApplyTerminal([]byte(terminalTemplate))
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 2 {
		t.Fatalf("changed = %v", changed)
	}

	got := decodeJSON(t, readFile(t, store))
	defaults := got["profiles"].(map[string]any)["defaults"].(map[string]any)
	if defaults["colorScheme"] != "Aether" || got["theme"] != "Aether" {
		t.Errorf("colorScheme = %v, theme = %v", defaults["colorScheme"], got["theme"])
	}
	schemes := got["schemes"].([]any)
	if len(schemes) != 1 || schemes[0].(map[string]any)["background"] != "#101010" {
		t.Errorf("schemes = %v", schemes)
	}
	if got["keybindings"] == nil || len(got["profiles"].(map[string]any)["list"].([]any)) != 2 {
		t.Error("other settings were lost")
	}
	if !strings.Contains(readFile(t, store), "    \"keybindings\": \n    [") {
		t.Error("formatting of untouched settings changed")
	}
	if readFile(t, store+OriginalBackupSuffix) != terminalFixture {
		t.Error("original backup does not hold the original file")
	}

	other := decodeJSON(t, readFile(t, unpackaged))
	if s := other["schemes"].([]any); len(s) != 2 || s[0].(map[string]any)["background"] != "#101010" || s[1].(map[string]any)["name"] != "Mine" {
		t.Errorf("existing Aether scheme not replaced in place: %v", s)
	}

	// A second apply is a no-op and keeps the original backup.
	changed, err = env.ApplyTerminal([]byte(terminalTemplate))
	if err != nil || len(changed) != 0 {
		t.Errorf("second apply: %v %v", changed, err)
	}
	if readFile(t, store+OriginalBackupSuffix) != terminalFixture {
		t.Error("original backup was overwritten")
	}
}

func TestApplyTerminalDefaultProfile(t *testing.T) {
	env, _, _, _ := testEnv(t)
	store := terminalFile(env)
	writeFile(t, store, terminalFixture)
	tpl := strings.Replace(terminalTemplate, `"applyTo": "defaults"`, `"applyTo": "default"`, 1)
	if _, err := env.ApplyTerminal([]byte(tpl)); err != nil {
		t.Fatal(err)
	}
	got := decodeJSON(t, readFile(t, store))
	list := got["profiles"].(map[string]any)["list"].([]any)
	if list[1].(map[string]any)["colorScheme"] != "Aether" || list[0].(map[string]any)["colorScheme"] != nil {
		t.Errorf("profiles = %v", list)
	}
	if _, ok := got["profiles"].(map[string]any)["defaults"].(map[string]any)["colorScheme"]; ok {
		t.Error("defaults changed in default-profile mode")
	}
}

func TestApplyTerminalRejectsBrokenSettings(t *testing.T) {
	env, _, _, _ := testEnv(t)
	store := terminalFile(env)
	writeFile(t, store, `{"profiles": [`)
	if _, err := env.ApplyTerminal([]byte(terminalTemplate)); err == nil {
		t.Fatal("broken settings.json accepted")
	}
	if readFile(t, store) != `{"profiles": [` {
		t.Error("broken settings.json was modified")
	}
	if _, err := env.ApplyTerminal([]byte(`{"scheme": {}}`)); err == nil {
		t.Error("scheme without a name accepted")
	}
}

func TestApplyTerminalWithoutTerminal(t *testing.T) {
	env, _, _, _ := testEnv(t)
	changed, err := env.ApplyTerminal([]byte(terminalTemplate))
	if err != nil || changed != nil {
		t.Errorf("no Windows Terminal: %v %v", changed, err)
	}
}

const leopardFixture = "# LeopardWM\r\n[layout]\r\ngap = 10\r\n\r\n[appearance]\r\nlanguage = \"en\"\r\n\r\n[hotkeys]\r\n\"Win+H\" = \"focus_left\"\r\n\r\n[[window_rules]]\r\nmatch_executable = \"vgtray.exe\"\r\naction = \"ignore\"\r\n"

const leopardTemplate = `[appearance]
active_border_color = "#a1b2c3"
tab_strip_bg = "101010"
tab_strip_opacity = 240
`

func TestApplyLeopardWM(t *testing.T) {
	env, _, _, runs := testEnv(t)
	file := filepath.Join(env.Paths.AppData, "leopardwm", "config", "config.toml")
	writeFile(t, file, leopardFixture)
	env.LookPath = func(name string) (string, error) {
		if name == "lwm" {
			return `C:\Program Files\LeopardWM\lwm.exe`, nil
		}
		return "", errors.New("not found")
	}

	changed, err := env.ApplyLeopardWM([]byte(leopardTemplate))
	if err != nil || !changed {
		t.Fatalf("ApplyLeopardWM = %v, %v", changed, err)
	}
	want := strings.Replace(leopardFixture, "language = \"en\"\r\n",
		"language = \"en\"\r\nactive_border_color = \"A1B2C3\"\r\ntab_strip_bg = \"101010\"\r\ntab_strip_opacity = 240\r\n", 1)
	if got := readFile(t, file); got != want {
		t.Errorf("config.toml =\n%q\nwant\n%q", got, want)
	}
	if !reflect.DeepEqual(*runs, [][]string{{`C:\Program Files\LeopardWM\lwm.exe`, "reload"}}) {
		t.Errorf("runs = %v", *runs)
	}
	if readFile(t, file+OriginalBackupSuffix) != leopardFixture {
		t.Error("missing original backup")
	}
}

func TestApplyLeopardWMValidatesColours(t *testing.T) {
	env, _, _, _ := testEnv(t)
	writeFile(t, filepath.Join(env.Paths.AppData, "leopardwm", "config", "config.toml"), leopardFixture)
	if _, err := env.ApplyLeopardWM([]byte("[appearance]\ntab_strip_bg = \"blue\"\n")); err == nil {
		t.Error("invalid colour accepted")
	}
	if _, err := env.ApplyLeopardWM([]byte("tab_strip_bg = \"101010\"\n")); err == nil {
		t.Error("template without [appearance] accepted")
	}
}

func TestApplyLeopardWMSkipsWithoutConfig(t *testing.T) {
	env, _, _, runs := testEnv(t)
	changed, err := env.ApplyLeopardWM([]byte(leopardTemplate))
	if err != nil || changed || len(*runs) != 0 {
		t.Errorf("without config: %v %v %v", changed, err, *runs)
	}
	// The Unix-style location is used when it is the only one.
	alt := filepath.Join(env.Paths.UserProfile, ".config", "leopardwm", "config.toml")
	writeFile(t, alt, "[appearance]\n")
	if env.Paths.LeopardWMConfigFile() != alt {
		t.Errorf("config file = %q", env.Paths.LeopardWMConfigFile())
	}
}

func TestApplyKomorebiMergesTheme(t *testing.T) {
	env, _, _, runs := testEnv(t)
	file := filepath.Join(env.Paths.UserProfile, "komorebi.json")
	writeFile(t, file, `{
  "border": true,
  "theme": {
    "palette": "Base16",
    "name": "Ashes",
    "unfocused_border": "Base03"
  },
  "ignore_rules": [
    { "kind": "Exe", "id": "vgtray.exe", "matching_strategy": "Equals" }
  ]
}
`)
	env.LookPath = func(string) (string, error) { return "komorebic.exe", nil }
	tpl := `// comment
{"theme": {"palette": "Custom", "name": null, "colours": {"base_00": "#101010"}}}`
	if _, err := env.ApplyKomorebi([]byte(tpl)); err != nil {
		t.Fatal(err)
	}
	got := decodeJSON(t, readFile(t, file))
	theme := got["theme"].(map[string]any)
	if theme["palette"] != "Custom" || theme["unfocused_border"] != "Base03" || theme["name"] != nil {
		t.Errorf("theme = %v", theme)
	}
	if !strings.Contains(readFile(t, file), `{ "kind": "Exe", "id": "vgtray.exe", "matching_strategy": "Equals" }`) {
		t.Error("untouched rules were reformatted")
	}
	if !reflect.DeepEqual(*runs, [][]string{{"komorebic.exe", "replace-configuration", file}}) {
		t.Errorf("runs = %v", *runs)
	}
}

func TestInstall(t *testing.T) {
	env, _, _, _ := testEnv(t)
	if dst, err := env.Install("alacritty", []byte("a")); err != nil || dst != "" {
		t.Errorf("alacritty without config dir: %q %v", dst, err)
	}
	if err := os.MkdirAll(filepath.Join(env.Paths.AppData, "alacritty"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst, err := env.Install("alacritty", []byte("[colors]\n"))
	if err != nil || dst != filepath.Join(env.Paths.AppData, "alacritty", "aether.toml") {
		t.Fatalf("Install = %q %v", dst, err)
	}
	// Aether replaces its own file without a backup...
	if _, err := env.Install("alacritty", []byte("[colors] # v2\n")); err != nil {
		t.Fatal(err)
	}
	if exists(dst + OriginalBackupSuffix) {
		t.Error("backed up Aether's own file")
	}
	// ...but keeps a copy of a file someone else wrote.
	writeFile(t, dst, "hand edited\n")
	if _, err := env.Install("alacritty", []byte("[colors] # v3\n")); err != nil {
		t.Fatal(err)
	}
	if readFile(t, dst+OriginalBackupSuffix) != "hand edited\n" || readFile(t, dst) != "[colors] # v3\n" {
		t.Error("foreign file was not backed up")
	}

	// WezTerm users with only ~/.wezterm.lua still get ~/.config/wezterm.
	writeFile(t, filepath.Join(env.Paths.UserProfile, ".wezterm.lua"), "return {}\n")
	if dst, err := env.Install(AppWezTerm, []byte("return {}\n")); err != nil || !strings.HasSuffix(dst, filepath.Join(".config", "wezterm", "aether.lua")) {
		t.Errorf("wezterm = %q %v", dst, err)
	}
}

func TestRestoreUndoesEveryTarget(t *testing.T) {
	env, reg, desk, _ := testEnv(t)
	original := filepath.Join(env.Paths.UserProfile, "img19.jpg")
	writeFile(t, original, "stock wallpaper")
	seedWindowsDefaults(t, reg, desk, original)
	before := registryDump(t, reg)

	store := terminalFile(env)
	writeFile(t, store, terminalFixture)
	leopard := filepath.Join(env.Paths.AppData, "leopardwm", "config", "config.toml")
	writeFile(t, leopard, leopardFixture)
	wall := filepath.Join(env.Paths.AppData, "aether", "theme", "backgrounds", "w.png")
	writeFile(t, wall, "aether wallpaper")

	if err := env.ApplySystem([]byte(`{"mode": "dark", "accent": "#e06c75", "accentOnTaskbar": true}`)); err != nil {
		t.Fatal(err)
	}
	if err := env.ApplyWallpaper(wall, []byte(`{"style": "fill", "desktopColor": "#101010"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ApplyTerminal([]byte(terminalTemplate)); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ApplyLeopardWM([]byte(leopardTemplate)); err != nil {
		t.Fatal(err)
	}
	// A second theme must not overwrite the recorded originals.
	if err := env.ApplySystem([]byte(`{"mode": "light", "accent": "#336699"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ApplyLeopardWM([]byte(strings.Replace(leopardTemplate, "101010", "202020", 1))); err != nil {
		t.Fatal(err)
	}

	result, err := env.Restore()
	if err != nil {
		t.Fatal(err)
	}
	if got := registryDump(t, reg); !reflect.DeepEqual(got, before) {
		t.Errorf("registry after restore:\n%v\nwant\n%v", got, before)
	}
	if desk.WallpaperSet != original {
		t.Errorf("wallpaper = %q, want %q", desk.WallpaperSet, original)
	}
	if desk.Color == nil || *desk.Color != [3]uint8{0, 0, 0} {
		t.Errorf("desktop colour = %v", desk.Color)
	}
	if got := readFile(t, leopard); got != leopardFixture {
		t.Errorf("config.toml after restore:\n%q\nwant\n%q", got, leopardFixture)
	}
	if got := decodeJSON(t, readFile(t, store)); !reflect.DeepEqual(got, decodeJSON(t, terminalFixture)) {
		t.Errorf("settings.json after restore:\n%s", readFile(t, store))
	}
	if len(result.Files) != 2 || result.Registry == 0 {
		t.Errorf("result = %+v", result)
	}
	if env.HasRestoreRecord() {
		t.Error("restore record left behind")
	}
	if _, err := env.Restore(); err == nil {
		t.Error("second restore succeeded")
	}
}

func TestPathsFromEnv(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\me`)
	t.Setenv("APPDATA", `C:\Users\me\AppData\Roaming`)
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("KOMOREBI_CONFIG_HOME", `D:\komorebi`)
	p := PathsFromEnv()
	if p.UserProfile != `C:\Users\me` || p.AppData != `C:\Users\me\AppData\Roaming` || p.KomorebiConfigHome != `D:\komorebi` {
		t.Errorf("paths = %+v", p)
	}
	if p.LocalAppData != filepath.Join(`C:\Users\me`, "AppData", "Local") {
		t.Errorf("LocalAppData fallback = %q", p.LocalAppData)
	}
	if p.KomorebiConfigFile() != filepath.Join(`D:\komorebi`, "komorebi.json") {
		t.Errorf("komorebi file = %q", p.KomorebiConfigFile())
	}
}

func TestInstallTargetsHonourXDGConfigHome(t *testing.T) {
	p := Paths{AppData: `C:\A`, LocalAppData: `C:\L`, UserProfile: `C:\U`}
	if got := p.InstallTargets()["neovim"].Dir; got != filepath.Join(`C:\L`, "nvim") {
		t.Errorf("neovim dir = %q", got)
	}
	p.XDGConfigHome = `C:\U\.config`
	if got := p.InstallTargets()["neovim"].Dir; got != filepath.Join(`C:\U\.config`, "nvim") {
		t.Errorf("neovim dir with XDG_CONFIG_HOME = %q", got)
	}
	if got := p.InstallTargets()[AppWezTerm].Dir; got != filepath.Join(`C:\U\.config`, "wezterm") {
		t.Errorf("wezterm dir with XDG_CONFIG_HOME = %q", got)
	}
}
