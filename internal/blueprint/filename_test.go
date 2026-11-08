package blueprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeFileStem(t *testing.T) {
	for name, want := range map[string]string{
		"midnight":        "midnight",
		`sub\name`:        "sub-name",
		"a/b":             "a-b",
		`..\..\escape`:    "-..-escape",
		"what?":           "what-",
		"CON":             "_CON",
		"com1":            "_com1",
		"  name.  ":       "name",
		"":                "",
		"../":             "",
		"trailing spaces": "trailing spaces",
	} {
		if got := safeFileStem(name); got != want {
			t.Errorf("safeFileStem(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSaveKeepsUnsafeNamesInTheBlueprintDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	service := NewService()

	for _, name := range []string{`sub\name`, "a/b", `..\..\escape`, "what?"} {
		bp := Blueprint{Name: name, Palette: validPaletteForTest()}
		if err := service.Save(name, bp); err != nil {
			t.Fatalf("Save(%q): %v", name, err)
		}
	}

	entries, err := os.ReadDir(service.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("Save created a subdirectory %q", entry.Name())
		}
	}
	loaded, err := service.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 4 {
		t.Errorf("LoadAll() returned %d blueprints, want 4", len(loaded))
	}
}

func TestExtractNameFromURLClearsQuery(t *testing.T) {
	for rawURL, want := range map[string]string{
		"https://example.com/wallpapers/theme.json":     "theme",
		"https://example.com/wallpapers/theme.json?x=1": "theme",
		"https://example.com/wallpapers/theme.yaml?x=1": "theme.yaml",
		"https://example.com/":                          "imported_blueprint",
		"https://example.com/dir/?token=a%2Fb":          "dir",
	} {
		if got := extractNameFromURL(rawURL); got != want {
			t.Errorf("extractNameFromURL(%q) = %q, want %q", rawURL, got, want)
		}
	}
}

func TestSaveImportedUsesAFileInTheBlueprintDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	bp := &Blueprint{Name: `sub\name?`, Palette: validPaletteForTest()}
	path, err := SaveImported(bp)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "aether", "blueprints") {
		t.Errorf("SaveImported path = %q", path)
	}
	if strings.ContainsAny(filepath.Base(path), `\/:*?"<>|`) {
		t.Errorf("SaveImported produced an invalid file name %q", filepath.Base(path))
	}
}

// validPaletteForTest returns a palette that passes validateBlueprint.
func validPaletteForTest() PaletteData {
	colors := make([]string, 16)
	for i := range colors {
		colors[i] = "#000000"
	}
	return PaletteData{Colors: colors, Wallpaper: "", LightMode: false}
}
