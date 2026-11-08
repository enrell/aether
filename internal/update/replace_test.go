package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "aether.exe")
	replacement := filepath.Join(t.TempDir(), "download.exe")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replaceExecutable(exe, replacement); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "new" {
		t.Errorf("executable = %q, want new", got)
	}
	if got, _ := os.ReadFile(exe + ".old"); string(got) != "old" {
		t.Errorf("moved-aside executable = %q, want old", got)
	}
	if _, err := os.Stat(exe + ".new"); !os.IsNotExist(err) {
		t.Error("staged file left behind")
	}
}

func TestReplaceExecutableMissingDownload(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "aether.exe")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceExecutable(exe, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing download accepted")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("executable changed to %q", got)
	}
}
