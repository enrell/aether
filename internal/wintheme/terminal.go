package wintheme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"aether/internal/jsonc"
)

// TerminalConfig is the rendered windows-terminal.json template.
type TerminalConfig struct {
	// ApplyTo selects the profiles that use the scheme: "defaults" (the
	// profiles.defaults section every profile inherits), "default" (the profile
	// named by defaultProfile) or "none".
	ApplyTo string          `json:"applyTo"`
	Scheme  json.RawMessage `json:"scheme"`
	// Theme is an optional window theme (tab row and tab colours).
	Theme json.RawMessage `json:"theme,omitempty"`
}

// TerminalSettingsFiles returns the settings.json of every installed Windows
// Terminal build: Store stable, Preview and Canary, and unpackaged installs.
func (p Paths) TerminalSettingsFiles() []string {
	var files []string
	for _, pkg := range []string{
		"Microsoft.WindowsTerminal_8wekyb3d8bbwe",
		"Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe",
		"Microsoft.WindowsTerminalCanary_8wekyb3d8bbwe",
	} {
		files = append(files, filepath.Join(p.LocalAppData, "Packages", pkg, "LocalState", "settings.json"))
	}
	files = append(files, filepath.Join(p.LocalAppData, "Microsoft", "Windows Terminal", "settings.json"))

	existing := files[:0]
	for _, f := range files {
		if exists(f) {
			existing = append(existing, f)
		}
	}
	return existing
}

// ApplyTerminal adds or updates the Aether colour scheme (and window theme)
// in every Windows Terminal settings.json and selects it. Windows Terminal
// reloads settings.json on its own. It returns the files it changed.
func (e *Env) ApplyTerminal(rendered []byte) ([]string, error) {
	var cfg TerminalConfig
	if err := decodeTemplate(rendered, &cfg); err != nil {
		return nil, fmt.Errorf("read windows-terminal.json template: %w", err)
	}
	schemeName, err := objectName(cfg.Scheme)
	if err != nil {
		return nil, fmt.Errorf("windows-terminal.json scheme: %w", err)
	}
	themeName := ""
	if len(cfg.Theme) > 0 && string(cfg.Theme) != "null" {
		if themeName, err = objectName(cfg.Theme); err != nil {
			return nil, fmt.Errorf("windows-terminal.json theme: %w", err)
		}
	}
	switch cfg.ApplyTo {
	case "", "defaults", "default", "none":
	default:
		return nil, fmt.Errorf("windows-terminal.json applyTo must be defaults, default or none, got %q", cfg.ApplyTo)
	}

	files := e.Paths.TerminalSettingsFiles()
	if len(files) == 0 {
		e.logf("Windows Terminal settings.json not found; skipping")
		return nil, nil
	}
	record, err := e.loadRestore()
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, file := range files {
		did, err := e.patchTerminal(record, file, cfg, schemeName, themeName)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", file, err)
		}
		if did {
			changed = append(changed, file)
			e.logf("Updated Windows Terminal scheme %q in %s", schemeName, file)
		}
	}
	return changed, nil
}

func (e *Env) patchTerminal(record *restoreRecord, file string, cfg TerminalConfig, schemeName, themeName string) (bool, error) {
	before, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	doc, err := jsonc.Parse(before)
	if err != nil {
		return false, fmt.Errorf("parse settings: %w", err)
	}

	if err := e.recordJSON(record, file, doc, "/schemes", schemeName); err != nil {
		return false, err
	}
	if err := upsertNamed(doc, "/schemes", schemeName, cfg.Scheme); err != nil {
		return false, err
	}
	if themeName != "" {
		if err := e.recordJSON(record, file, doc, "/themes", themeName); err != nil {
			return false, err
		}
		if err := upsertNamed(doc, "/themes", themeName, cfg.Theme); err != nil {
			return false, err
		}
		if err := e.recordJSON(record, file, doc, "/theme", ""); err != nil {
			return false, err
		}
		if err := doc.Set("/theme", quote(themeName)); err != nil {
			return false, err
		}
	}

	if target, err := terminalProfilePointer(doc, cfg.ApplyTo); err != nil {
		return false, err
	} else if target != "" {
		if err := e.recordJSON(record, file, doc, target+"/colorScheme", ""); err != nil {
			return false, err
		}
		if err := doc.Set(target+"/colorScheme", quote(schemeName)); err != nil {
			return false, err
		}
	}
	return writeConfig(file, before, doc.Bytes())
}

// terminalProfilePointer returns the pointer of the profile object that
// should use the scheme, creating profiles.defaults when needed.
func terminalProfilePointer(doc *jsonc.Doc, applyTo string) (string, error) {
	switch applyTo {
	case "none":
		return "", nil
	case "default":
		guid, _ := doc.GetString("/defaultProfile")
		if guid != "" {
			list := "/profiles/list"
			if doc.Kind("/profiles") == '[' {
				list = "/profiles" // pre-1.0 settings kept profiles as an array
			}
			if i := doc.FindBy(list, "guid", guid); i >= 0 {
				return list + "/" + strconv.Itoa(i), nil
			}
		}
		// The default profile may be generated at runtime and absent from
		// the file; fall back to the defaults every profile inherits.
	}
	if doc.Kind("/profiles") == '[' {
		return "", fmt.Errorf("settings use the legacy profiles array; set applyTo to \"default\" or \"none\"")
	}
	if err := doc.Ensure("/profiles", "{}"); err != nil {
		return "", err
	}
	if err := doc.Ensure("/profiles/defaults", "{}"); err != nil {
		return "", err
	}
	return "/profiles/defaults", nil
}

// upsertNamed replaces the item called name in the array at ptr, or appends
// it.
func upsertNamed(doc *jsonc.Doc, ptr, name string, item []byte) error {
	if err := doc.Ensure(ptr, "[]"); err != nil {
		return err
	}
	if i := doc.FindByName(ptr, name); i >= 0 {
		return doc.Set(ptr+"/"+strconv.Itoa(i), item)
	}
	return doc.Append(ptr, item)
}

func objectName(raw json.RawMessage) (string, error) {
	var v struct {
		Name string `json:"name"`
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("missing")
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	if v.Name == "" {
		return "", fmt.Errorf("needs a name")
	}
	return v.Name, nil
}

func quote(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}
