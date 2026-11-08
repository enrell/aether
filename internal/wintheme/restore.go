package wintheme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aether/internal/jsonc"
	"aether/internal/platform"
	"aether/internal/winapi"
)

// The restore record keeps the value each setting had before Aether first
// changed it, so `aether --restore-windows` can undo every Windows target.
// Later applies never overwrite a recorded value.

const restoreFile = "windows-restore.json"

type restoreRecord struct {
	Version   int             `json:"version"`
	Created   time.Time       `json:"created"`
	Registry  []registryEntry `json:"registry,omitempty"`
	Wallpaper *wallpaperEntry `json:"wallpaper,omitempty"`
	JSON      []jsonEntry     `json:"json,omitempty"`
	TOML      []tomlEntry     `json:"toml,omitempty"`
}

type registryEntry struct {
	Key     string       `json:"key"`
	Name    string       `json:"name"`
	Existed bool         `json:"existed"`
	Value   winapi.Value `json:"value"`
}

type wallpaperEntry struct {
	Path string `json:"path"`
	// Copy is Aether's copy of the image, kept because Windows may overwrite
	// its own cached wallpaper file.
	Copy string `json:"copy,omitempty"`
}

// jsonEntry records the previous value at Pointer in File, or the previous
// array item called ItemName in the array at Pointer.
type jsonEntry struct {
	File     string          `json:"file"`
	Pointer  string          `json:"pointer"`
	ItemName string          `json:"itemName,omitempty"`
	Existed  bool            `json:"existed"`
	Value    json.RawMessage `json:"value,omitempty"`
}

func (e *Env) restorePath() string { return filepath.Join(e.StateDir, restoreFile) }

func (e *Env) loadRestore() (*restoreRecord, error) {
	data, err := os.ReadFile(e.restorePath())
	if os.IsNotExist(err) {
		return &restoreRecord{Version: 1, Created: e.now().UTC()}, nil
	}
	if err != nil {
		return nil, err
	}
	var record restoreRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("read %s: %w", e.restorePath(), err)
	}
	return &record, nil
}

func (e *Env) saveRestore(record *restoreRecord) error {
	if e.StateDir == "" {
		return errors.New("no state directory for the restore record")
	}
	return platform.WriteJSON(e.restorePath(), record)
}

// HasRestoreRecord reports whether there is anything to restore.
func (e *Env) HasRestoreRecord() bool { return exists(e.restorePath()) }

func (r *restoreRecord) hasRegistry(key, name string) bool {
	for _, entry := range r.Registry {
		if strings.EqualFold(entry.Key, key) && strings.EqualFold(entry.Name, name) {
			return true
		}
	}
	return false
}

func (r *restoreRecord) hasJSON(file, pointer, item string) bool {
	for _, entry := range r.JSON {
		if strings.EqualFold(entry.File, file) && entry.Pointer == pointer && entry.ItemName == item {
			return true
		}
	}
	return false
}

// setRegistry writes a value after recording the one it replaces. It
// returns false when the value already had the requested data.
func (e *Env) setRegistry(record *restoreRecord, key, name string, value winapi.Value) (bool, error) {
	current, ok, err := e.System.Registry.Get(key, name)
	if err != nil {
		return false, err
	}
	if ok && current.Equal(value) {
		return false, nil
	}
	if !record.hasRegistry(key, name) {
		record.Registry = append(record.Registry, registryEntry{Key: key, Name: name, Existed: ok, Value: current})
		// Persist before writing so a crash never loses the original value.
		if err := e.saveRestore(record); err != nil {
			return false, err
		}
	}
	return true, e.System.Registry.Set(key, name, value)
}

// recordJSON remembers the value at pointer (or the named item of the array
// at pointer) before Aether changes it.
func (e *Env) recordJSON(record *restoreRecord, file string, doc *jsonc.Doc, pointer, item string) error {
	if record.hasJSON(file, pointer, item) {
		return nil
	}
	entry := jsonEntry{File: file, Pointer: pointer, ItemName: item}
	if item != "" {
		if i := doc.FindByName(pointer, item); i >= 0 {
			entry.Value, entry.Existed = doc.Get(pointer + "/" + strconv.Itoa(i))
		}
	} else {
		entry.Value, entry.Existed = doc.Get(pointer)
	}
	record.JSON = append(record.JSON, entry)
	return e.saveRestore(record)
}

// RestoreResult lists what Restore put back.
type RestoreResult struct {
	Registry  int
	Wallpaper string
	Files     []string
	Reload    []string
}

// reloadAfterRestore asks the window managers whose files were restored to
// load them again.
func (e *Env) reloadAfterRestore(files []string) []string {
	var reloaded []string
	for _, f := range files {
		switch {
		case strings.EqualFold(f, e.Paths.KomorebiConfigFile()):
			e.reloadKomorebi(f)
			reloaded = append(reloaded, AppKomorebi)
		case strings.EqualFold(f, e.Paths.LeopardWMConfigFile()):
			e.reloadLeopardWM()
			reloaded = append(reloaded, AppLeopardWM)
		}
	}
	return reloaded
}

// Restore puts back every setting recorded before Aether first changed it and
// removes the record. Configuration files keep their backups.
func (e *Env) Restore() (RestoreResult, error) {
	var result RestoreResult
	if !e.HasRestoreRecord() {
		return result, errors.New("nothing to restore: Aether has not changed Windows settings")
	}
	record, err := e.loadRestore()
	if err != nil {
		return result, err
	}

	var errs []error
	if len(record.Registry) > 0 || record.Wallpaper != nil {
		if err := e.requireSystem(); err != nil {
			return result, err
		}
	}
	for i := len(record.Registry) - 1; i >= 0; i-- {
		entry := record.Registry[i]
		var err error
		if entry.Existed {
			err = e.System.Registry.Set(entry.Key, entry.Name, entry.Value)
		} else {
			err = e.System.Registry.Delete(entry.Key, entry.Name)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result.Registry++
		if strings.EqualFold(entry.Key, desktopColorsKey) && strings.EqualFold(entry.Name, "Background") && entry.Existed {
			if r, g, b, ok := parseRGBTriplet(entry.Value.String); ok {
				_ = e.System.Desktop.SetDesktopColor(r, g, b)
			}
		}
	}

	if wp := record.Wallpaper; wp != nil {
		path := wp.Path
		if wp.Copy != "" && exists(wp.Copy) && (path == "" || !exists(path) || isWindowsTranscoded(path)) {
			path = wp.Copy
		}
		if path != "" {
			if err := e.System.Desktop.SetWallpaper(path); err != nil {
				errs = append(errs, err)
			} else {
				result.Wallpaper = path
			}
		}
	}

	byFile := map[string][]jsonEntry{}
	var order []string
	for _, entry := range record.JSON {
		if _, seen := byFile[entry.File]; !seen {
			order = append(order, entry.File)
		}
		byFile[entry.File] = append(byFile[entry.File], entry)
	}
	for _, file := range order {
		if err := restoreJSONFile(file, byFile[file]); err != nil {
			errs = append(errs, err)
			continue
		}
		result.Files = append(result.Files, file)
	}

	tomlByFile := map[string][]tomlEntry{}
	var tomlOrder []string
	for _, entry := range record.TOML {
		if _, seen := tomlByFile[entry.File]; !seen {
			tomlOrder = append(tomlOrder, entry.File)
		}
		tomlByFile[entry.File] = append(tomlByFile[entry.File], entry)
	}
	for _, file := range tomlOrder {
		if err := restoreTOMLFile(file, tomlByFile[file]); err != nil {
			errs = append(errs, err)
			continue
		}
		result.Files = append(result.Files, file)
	}
	result.Reload = e.reloadAfterRestore(result.Files)

	if len(record.Registry) > 0 {
		if err := e.System.Desktop.BroadcastSettingChange("ImmersiveColorSet"); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}
	return result, os.Remove(e.restorePath())
}

func restoreJSONFile(file string, entries []jsonEntry) error {
	before, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return nil // the application config was removed; nothing to restore
	}
	if err != nil {
		return err
	}
	doc, err := jsonc.Parse(before)
	if err != nil {
		return fmt.Errorf("parse %s: %w", file, err)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		pointer := entry.Pointer
		if entry.ItemName != "" {
			idx := doc.FindByName(entry.Pointer, entry.ItemName)
			switch {
			case idx >= 0 && entry.Existed:
				err = doc.Set(entry.Pointer+"/"+strconv.Itoa(idx), entry.Value)
			case idx >= 0:
				err = doc.Delete(entry.Pointer + "/" + strconv.Itoa(idx))
			case entry.Existed && doc.Kind(entry.Pointer) == '[':
				err = doc.Append(entry.Pointer, entry.Value)
			}
		} else if entry.Existed {
			err = doc.Set(pointer, entry.Value)
		} else {
			err = doc.Delete(pointer)
		}
		if err != nil {
			return fmt.Errorf("restore %s%s: %w", file, pointer, err)
		}
	}
	_, err = writeConfig(file, before, doc.Bytes())
	return err
}

func parseRGBTriplet(s string) (r, g, b uint8, ok bool) {
	parts := strings.Fields(s)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var v [3]uint8
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			return 0, 0, 0, false
		}
		v[i] = uint8(n)
	}
	return v[0], v[1], v[2], true
}

// isWindowsTranscoded reports whether path is Windows' own wallpaper cache,
// which is overwritten whenever the wallpaper changes.
func isWindowsTranscoded(path string) bool {
	return strings.EqualFold(filepath.Base(path), "TranscodedWallpaper")
}
