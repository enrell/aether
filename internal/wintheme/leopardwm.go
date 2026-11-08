package wintheme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"aether/internal/tomledit"

	"github.com/BurntSushi/toml"
)

const leopardTable = "appearance"

var leopardHex = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// LeopardWMConfigFile returns the config.toml LeopardWM loads: the first of
// %APPDATA%\leopardwm\config\config.toml and ~\.config\leopardwm\config.toml
// that exists, or "" when LeopardWM has not been set up.
func (p Paths) LeopardWMConfigFile() string {
	for _, f := range []string{
		filepath.Join(p.AppData, "leopardwm", "config", "config.toml"),
		filepath.Join(p.UserProfile, ".config", "leopardwm", "config.toml"),
	} {
		if exists(f) {
			return f
		}
	}
	return ""
}

// leopardChanges reads the [appearance] keys of the rendered template in
// template order.
func leopardChanges(rendered []byte) ([]tomledit.Change, error) {
	var tpl map[string]any
	md, err := toml.Decode(string(rendered), &tpl)
	if err != nil {
		return nil, fmt.Errorf("read leopardwm.toml template: %w", err)
	}
	table, ok := tpl[leopardTable].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("leopardwm.toml template has no [%s] table", leopardTable)
	}
	var changes []tomledit.Change
	for _, key := range md.Keys() {
		if len(key) != 2 || key[0] != leopardTable {
			continue
		}
		name := key[1]
		var literal string
		switch v := table[name].(type) {
		case string:
			if isLeopardColorKey(name) {
				v = strings.TrimPrefix(v, "#") // LeopardWM rejects the '#'
				if !leopardHex.MatchString(v) {
					return nil, fmt.Errorf("leopardwm.toml %s = %q is not six hex digits", name, v)
				}
				v = strings.ToUpper(v)
			}
			literal = tomledit.String(v)
		case int64:
			literal = strconv.FormatInt(v, 10)
		case bool:
			literal = strconv.FormatBool(v)
		case float64:
			literal = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			return nil, fmt.Errorf("leopardwm.toml %s has an unsupported value type", name)
		}
		changes = append(changes, tomledit.Change{Key: name, Value: literal})
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("leopardwm.toml template sets no [%s] keys", leopardTable)
	}
	return changes, nil
}

func isLeopardColorKey(key string) bool {
	return strings.HasSuffix(key, "_color") || strings.HasSuffix(key, "_bg") || strings.HasSuffix(key, "_text")
}

// ApplyLeopardWM sets the [appearance] colours of LeopardWM's config.toml and
// runs `lwm reload`. Every other line of the file is kept as it is. It does
// nothing when LeopardWM has no config file.
func (e *Env) ApplyLeopardWM(rendered []byte) (bool, error) {
	changes, err := leopardChanges(rendered)
	if err != nil {
		return false, err
	}
	file := e.Paths.LeopardWMConfigFile()
	if file == "" {
		e.logf("LeopardWM config.toml not found; skipping")
		return false, nil
	}
	before, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	record, err := e.loadRestore()
	if err != nil {
		return false, err
	}
	for _, c := range changes {
		if record.hasTOML(file, leopardTable, c.Key) {
			continue
		}
		raw, ok, err := tomledit.Get(before, leopardTable, c.Key)
		if err != nil {
			return false, fmt.Errorf("%s: %w", file, err)
		}
		record.TOML = append(record.TOML, tomlEntry{File: file, Table: leopardTable, Key: c.Key, Existed: ok, Value: raw})
		if err := e.saveRestore(record); err != nil {
			return false, err
		}
	}
	after, err := tomledit.Edit(before, leopardTable, changes)
	if err != nil {
		return false, fmt.Errorf("%s: %w", file, err)
	}
	changed, err := writeConfig(file, before, after)
	if err != nil || !changed {
		return changed, err
	}
	e.logf("Updated LeopardWM [appearance] colours in %s", file)
	e.reloadLeopardWM()
	return true, nil
}

func (e *Env) reloadLeopardWM() {
	if e.LookPath == nil || e.Run == nil {
		return
	}
	for _, name := range []string{"lwm", "leopardwm-cli"} {
		cli, err := e.LookPath(name)
		if err != nil {
			continue
		}
		if err := e.Run(cli, "reload"); err != nil {
			// LeopardWM is probably not running; it reads the file on start.
			e.logf("lwm reload: %v", err)
		}
		return
	}
	e.logf("lwm not found on PATH; LeopardWM will use the new colours after it restarts")
}

// tomlEntry records a TOML key's literal before Aether changed it.
type tomlEntry struct {
	File    string `json:"file"`
	Table   string `json:"table"`
	Key     string `json:"key"`
	Existed bool   `json:"existed"`
	Value   string `json:"value,omitempty"`
}

func (r *restoreRecord) hasTOML(file, table, key string) bool {
	for _, entry := range r.TOML {
		if strings.EqualFold(entry.File, file) && entry.Table == table && entry.Key == key {
			return true
		}
	}
	return false
}

func restoreTOMLFile(file string, entries []tomlEntry) error {
	before, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	byTable := map[string][]tomledit.Change{}
	var tables []string
	for _, entry := range entries {
		if _, seen := byTable[entry.Table]; !seen {
			tables = append(tables, entry.Table)
		}
		byTable[entry.Table] = append(byTable[entry.Table], tomledit.Change{Key: entry.Key, Value: entry.Value, Delete: !entry.Existed})
	}
	after := before
	for _, table := range tables {
		if after, err = tomledit.Edit(after, table, byTable[table]); err != nil {
			return fmt.Errorf("restore %s: %w", file, err)
		}
	}
	_, err = writeConfig(file, before, after)
	return err
}
