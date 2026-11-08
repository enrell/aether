package main

import (
	"aether/internal/testenv"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetSettingsIncludesDefaultWallpaperFolder(t *testing.T) {
	configHome := t.TempDir()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", home)
	testenv.WindowsProfile(t, home)

	settings := NewApp().GetSettings()
	got, ok := settings[wallpaperFolderSetting].(string)
	if !ok {
		t.Fatalf("wallpaper folder type = %T; want string", settings[wallpaperFolderSetting])
	}
	want := filepath.Join(home, "Wallpapers")
	if runtime.GOOS == "windows" {
		want = filepath.Join(home, "Pictures", "Wallpapers")
	}
	if got != want {
		t.Errorf("wallpaper folder = %q; want %q", got, want)
	}
}

func TestReadImageAsDataURLSniffsExtensionlessFile(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "downloaded")
	// Minimal 1x1 PNG.
	png := []byte{
		0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
		0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
		0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
		0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(imagePath, png, 0o600); err != nil {
		t.Fatal(err)
	}
	dataURL, err := NewApp().ReadImageAsDataURL(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Errorf("data URL = %.40q", dataURL)
	}

	textPath := filepath.Join(dir, "notes")
	if err := os.WriteFile(textPath, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewApp().ReadImageAsDataURL(textPath); err == nil {
		t.Error("ReadImageAsDataURL accepted a text file")
	}
}

func TestExistingAdditionalImagesDropsMissingEntries(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "kept.jpg")
	if err := os.WriteFile(kept, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	notImage := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notImage, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := existingAdditionalImages([]string{
		kept,
		filepath.Join(dir, "missing.jpg"),
		notImage,
		"",
	})
	if len(got) != 1 || got[0] != kept {
		t.Errorf("existingAdditionalImages = %v, want [%q]", got, kept)
	}
}

func TestFolderToShowFindsContainingFolder(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "favorites.zip")
	if err := os.WriteFile(file, []byte("zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := folderToShow(file); err != nil || got != dir {
		t.Errorf("folderToShow(file) = %q, %v; want %q", got, err, dir)
	}
	if got, err := folderToShow(dir); err != nil || got != dir {
		t.Errorf("folderToShow(dir) = %q, %v; want %q", got, err, dir)
	}
	if _, err := folderToShow(filepath.Join(dir, "missing.zip")); err == nil {
		t.Error("folderToShow accepted a missing file")
	}
	if _, err := folderToShow("  "); err == nil {
		t.Error("folderToShow accepted an empty path")
	}
}

func TestScanLocalWallpapersUsesConfiguredFolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	folder := t.TempDir()
	nested := filepath.Join(folder, "landscapes")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(nested, "mountain.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	if err := app.SaveSettings(map[string]interface{}{
		wallpaperFolderSetting: folder,
	}); err != nil {
		t.Fatal(err)
	}

	wallpapers, err := app.ScanLocalWallpapers()
	if err != nil {
		t.Fatal(err)
	}
	if len(wallpapers) != 1 {
		t.Fatalf("wallpaper count = %d; want 1", len(wallpapers))
	}
	if wallpapers[0].Path != imagePath {
		t.Errorf("wallpaper path = %q; want %q", wallpapers[0].Path, imagePath)
	}
	wantName := filepath.Join("landscapes", "mountain.png")
	if wallpapers[0].Name != wantName {
		t.Errorf("wallpaper name = %q; want %q", wallpapers[0].Name, wantName)
	}
}

func TestScanLocalWallpapersReportsMissingConfiguredFolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	app := NewApp()
	missing := filepath.Join(t.TempDir(), "missing")
	if err := app.SaveSettings(map[string]interface{}{
		wallpaperFolderSetting: missing,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := app.ScanLocalWallpapers(); err == nil {
		t.Fatal("ScanLocalWallpapers() error = nil; want missing folder error")
	}
}
