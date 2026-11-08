package wintheme

import (
	"fmt"
	"os"
	"path/filepath"

	"aether/internal/jsonc"
)

// KomorebiConfigFile returns komorebi.json in %KOMOREBI_CONFIG_HOME% or the
// user profile.
func (p Paths) KomorebiConfigFile() string {
	dir := p.KomorebiConfigHome
	if dir == "" {
		dir = p.UserProfile
	}
	return filepath.Join(dir, "komorebi.json")
}

// ApplyKomorebi merges the rendered komorebi.json template into the user's
// komorebi.json as a JSON merge patch, then asks a running komorebi to load
// it with `komorebic replace-configuration`. It does nothing when the user
// has no komorebi.json.
func (e *Env) ApplyKomorebi(rendered []byte) (bool, error) {
	file := e.Paths.KomorebiConfigFile()
	before, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		e.logf("komorebi.json not found at %s; skipping", file)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	doc, err := jsonc.Parse(before)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", file, err)
	}
	patch, err := jsonc.Parse(rendered)
	if err != nil {
		return false, fmt.Errorf("read komorebi.json template: %w", err)
	}
	record, err := e.loadRestore()
	if err != nil {
		return false, err
	}
	for _, key := range patch.Keys("") {
		if err := e.recordJSON(record, file, doc, "/"+jsonc.EscapePointer(key), ""); err != nil {
			return false, err
		}
	}
	if err := doc.MergePatch("", rendered); err != nil {
		return false, fmt.Errorf("merge into %s: %w", file, err)
	}
	changed, err := writeConfig(file, before, doc.Bytes())
	if err != nil || !changed {
		return changed, err
	}
	e.logf("Updated komorebi theme in %s", file)
	e.reloadKomorebi(file)
	return true, nil
}

func (e *Env) reloadKomorebi(file string) {
	if e.LookPath == nil || e.Run == nil {
		return
	}
	komorebic, err := e.LookPath("komorebic")
	if err != nil {
		e.logf("komorebic not found; komorebi will use the new colours after it restarts")
		return
	}
	if err := e.Run(komorebic, "replace-configuration", file); err != nil {
		// komorebi is probably not running; it reads the file on start.
		e.logf("komorebic replace-configuration: %v", err)
	}
}

// InstallTarget describes where a cross-platform app reads Aether's theme on
// Windows.
type InstallTarget struct {
	App string
	// Dir must exist for the target to be installed: Aether does not create
	// configuration for applications that are not set up.
	Dir string
	// File is the generated file inside Dir.
	File string
	// Hint tells the user how to load the file.
	Hint string
}

// InstallTargets returns the Windows install locations of cross-platform
// apps whose theme is a standalone file.
func (p Paths) InstallTargets() map[string]InstallTarget {
	wezDir := filepath.Join(p.UserProfile, ".config", "wezterm")
	nvimDir := filepath.Join(p.LocalAppData, "nvim")
	if p.XDGConfigHome != "" {
		wezDir = filepath.Join(p.XDGConfigHome, "wezterm")
		nvimDir = filepath.Join(p.XDGConfigHome, "nvim")
	}
	return map[string]InstallTarget{
		"alacritty": {
			App:  "alacritty",
			Dir:  filepath.Join(p.AppData, "alacritty"),
			File: "aether.toml",
			Hint: `add import = ["~/AppData/Roaming/alacritty/aether.toml"] under [general] in alacritty.toml`,
		},
		AppWezTerm: {
			App:  AppWezTerm,
			Dir:  wezDir,
			File: "aether.lua",
			Hint: `add config.colors = require("aether") to your wezterm config`,
		},
		"neovim": {
			App:  "neovim",
			Dir:  nvimDir,
			File: filepath.Join("lua", "plugins", "aether.lua"),
			Hint: "lazy.nvim loads lua/plugins/aether.lua when your config imports the plugins folder",
		},
	}
}

// Install writes a rendered template for a cross-platform app to its Windows
// location. It returns the path written, or "" when the app is not set up.
func (e *Env) Install(app string, rendered []byte) (string, error) {
	target, ok := e.Paths.InstallTargets()[app]
	if !ok {
		return "", fmt.Errorf("no Windows install location for %s", app)
	}
	dir := target.Dir
	if app == AppWezTerm && !isDir(dir) && exists(filepath.Join(e.Paths.UserProfile, ".wezterm.lua")) {
		// ~/.wezterm.lua users still get ~/.config/wezterm on require's path.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if !isDir(dir) {
		e.logf("%s is not set up (%s does not exist); skipping", app, dir)
		return "", nil
	}
	dst := filepath.Join(dir, target.File)
	if err := e.installFile(dst, rendered); err != nil {
		return "", err
	}
	e.logf("Installed %s theme to %s (%s)", app, dst, target.Hint)
	return dst, nil
}
