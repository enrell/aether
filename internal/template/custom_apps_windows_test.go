//go:build windows

package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessCustomAppsRunsPowerShellHook(t *testing.T) {
	customDir, themeDir, _ := setupCustomAppsTest(t)
	destination := filepath.Join(t.TempDir(), "user.conf")
	appDir := writeCustomApp(t, customDir, "testapp", `%TEMP_DEST%`, false)
	t.Setenv("TEMP_DEST", destination)
	marker := filepath.Join(t.TempDir(), "hook-ran")
	hook := "Set-Content -LiteralPath '" + marker + "' -Value 'ok'\n"
	if err := os.WriteFile(filepath.Join(appDir, "post-apply.ps1"), []byte(hook), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ProcessCustomApps(themeDir, map[string]string{"background": "#123456"}); err != nil {
		t.Fatal(err)
	}
	// %TEMP_DEST% expanded; the destination is a symlink or, without the
	// privilege, a copy.
	if data, err := os.ReadFile(destination); err != nil || string(data) != "bg=#123456\n" {
		t.Fatalf("destination = %q, %v", data, err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		data, err := os.ReadFile(marker)
		if err == nil && strings.TrimSpace(string(data)) == "ok" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("post-apply.ps1 did not run: %q, %v", data, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestExpandWindowsEnv(t *testing.T) {
	t.Setenv("AETHER_TEST_DIR", `C:\cfg`)
	for in, want := range map[string]string{
		`%AETHER_TEST_DIR%\cava\theme`: `C:\cfg\cava\theme`,
		`%UNSET_AETHER_VAR%\x`:         `%UNSET_AETHER_VAR%\x`,
		`100%`:                         `100%`,
	} {
		if got := expandWindowsEnv(in); got != want {
			t.Errorf("expandWindowsEnv(%q) = %q, want %q", in, got, want)
		}
	}
}
