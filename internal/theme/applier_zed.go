package theme

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"aether/internal/platform"
)

// zedOutputFilename is the name used for the installed Zed theme file.
const zedOutputFilename = "aether.json"

// zedSourceFilename is the name of the Zed template file in the theme directory.
const zedSourceFilename = "aether.zed.json"

// zedThemesDir returns ~/.config/zed/themes, or %APPDATA%\Zed\themes on
// Windows.
func zedThemesDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			dir, err := os.UserConfigDir()
			if err != nil {
				return "", err
			}
			appData = dir
		}
		return filepath.Join(appData, "Zed", "themes"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zed", "themes"), nil
}

// ApplyZedTheme copies the generated aether.zed.json from themeDir to
// Zed's user themes directory as aether.json.
func ApplyZedTheme(themeDir string) error {
	sourcePath := filepath.Join(themeDir, zedSourceFilename)

	if !platform.FileExists(sourcePath) {
		log.Printf("%s not found in theme directory, skipping Zed theme application", zedSourceFilename)
		return nil
	}

	zedThemesDir, err := zedThemesDir()
	if err != nil {
		return err
	}
	if err := platform.EnsureDir(zedThemesDir); err != nil {
		return err
	}

	destPath := filepath.Join(zedThemesDir, zedOutputFilename)
	if err := platform.CopyFile(sourcePath, destPath); err != nil {
		return err
	}

	log.Printf("Copied Zed theme to: %s", destPath)
	return nil
}
