package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"aether/internal/platform"
)

// customAppConfig represents the config.json structure inside a custom app folder.
type customAppConfig struct {
	Template    string `json:"template"`
	Destination string `json:"destination"`
}

// ProcessCustomApps scans ~/.config/aether/custom/ for app-specific templates,
// processes each template with variable substitution, writes the output to
// themeDir, creates destination symlinks, and runs any post-apply.sh scripts.
// App failures are collected and returned; failed installs never run their hook.
//
// Directory structure expected:
//
//	~/.config/aether/custom/
//	  cava/
//	    config.json      -> {"template": "theme.ini", "destination": "~/.config/cava/themes/aether"}
//	    theme.ini        -> Template file with {color} variables
//	    post-apply.sh    -> Optional script to run after apply
func ProcessCustomApps(themeDir string, variables map[string]string) error {
	customDir := platform.CustomDir()

	entries, err := os.ReadDir(customDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appName := entry.Name()
		appPath := filepath.Join(customDir, appName)

		if err := processCustomApp(appPath, appName, themeDir, variables); err != nil {
			errs = append(errs, fmt.Errorf("custom app %q: %w", appName, err))
		}
	}

	return errors.Join(errs...)
}

// processCustomApp handles a single custom app directory: reads config.json,
// processes the template, writes output, creates symlink, and runs post-apply.
func processCustomApp(appPath, appName, themeDir string, variables map[string]string) error {
	// Read config.json
	configPath := filepath.Join(appPath, "config.json")
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		log.Printf("[%s] Missing config.json, skipping", appName)
		return nil
	}
	if err != nil {
		return err
	}

	var config customAppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if config.Template == "" {
		return fmt.Errorf("config.json missing 'template' field")
	}

	// Read and process the template
	templatePath := filepath.Join(appPath, config.Template)
	content, err := platform.ReadText(templatePath)
	if err != nil {
		return err
	}

	processed := ProcessTemplate(content, variables)

	// Write processed output to theme directory
	outputFileName := appName + "-" + config.Template
	outputPath, err := filepath.Abs(filepath.Join(themeDir, outputFileName))
	if err != nil {
		return err
	}
	previous, _ := os.ReadFile(outputPath)
	if err := platform.WriteText(outputPath, processed); err != nil {
		return err
	}
	log.Printf("[%s] Processed template: %s", appName, config.Template)

	// Create destination symlink if configured
	if config.Destination != "" {
		destPath := expandHome(config.Destination)
		if err := installDestination(outputPath, destPath, previous); err != nil {
			return fmt.Errorf("install symlink at %s: %w", destPath, err)
		}
		log.Printf("[%s] Symlinked -> %s", appName, destPath)
	}

	// Run the post-apply hook if it exists
	if hook, args := postApplyHook(appPath); hook != "" {
		if err := platform.RunAsync(args[0], args[1:]...); err != nil {
			return fmt.Errorf("start %s: %w", filepath.Base(hook), err)
		}
		log.Printf("[%s] Executed %s", appName, filepath.Base(hook))
	}

	return nil
}

// postApplyHook returns the app's post-apply script and the command that
// runs it. Windows also accepts post-apply.ps1, .cmd and .bat; a .sh hook
// runs there only when bash (Git Bash, MSYS2) is on PATH.
func postApplyHook(appPath string) (string, []string) {
	if runtime.GOOS == "windows" {
		for _, hook := range []struct {
			file string
			cmd  []string
		}{
			{"post-apply.ps1", []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File"}},
			{"post-apply.cmd", []string{"cmd.exe", "/c"}},
			{"post-apply.bat", []string{"cmd.exe", "/c"}},
		} {
			path := filepath.Join(appPath, hook.file)
			if platform.FileExists(path) {
				return path, append(hook.cmd, path)
			}
		}
		if !platform.CommandExists("bash") {
			return "", nil
		}
	}
	path := filepath.Join(appPath, "post-apply.sh")
	if platform.FileExists(path) {
		return path, []string{"bash", path}
	}
	return "", nil
}

// installDestination links dest to the generated output. Windows only allows
// symlinks with Developer Mode or elevation; without them Aether copies the
// output instead, replacing dest only when it still holds the previous copy.
func installDestination(output, dest string, previous []byte) error {
	err := platform.CreateSymlink(output, dest)
	if err == nil || runtime.GOOS != "windows" || !isSymlinkPrivilegeError(err) && !errors.Is(err, os.ErrExist) {
		return err
	}
	if info, statErr := os.Lstat(dest); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return err
		}
		current, readErr := os.ReadFile(dest)
		if readErr != nil {
			return readErr
		}
		if previous == nil || !bytes.Equal(current, previous) {
			return err // not a copy Aether made
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	} else if !isSymlinkPrivilegeError(err) {
		return err
	}
	return platform.CopyFile(output, dest)
}

// isSymlinkPrivilegeError reports ERROR_PRIVILEGE_NOT_HELD (1314).
func isSymlinkPrivilegeError(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == 1314
}

// ReadCustomOverride checks if a custom override template exists for the given
// file name in ~/.config/aether/custom/. If found, returns its content and true.
// If not found, returns empty string and false.
func ReadCustomOverride(fileName string) (string, bool) {
	customPath := filepath.Join(platform.CustomDir(), fileName)
	if !platform.FileExists(customPath) {
		return "", false
	}
	content, err := platform.ReadText(customPath)
	if err != nil {
		log.Printf("Error reading custom override %s: %v", fileName, err)
		return "", false
	}
	log.Printf("Using custom override for %s", fileName)
	return content, true
}

// expandHome replaces a leading "~/" with the user's home directory. On
// Windows it also expands %VARIABLE% references such as %APPDATA%.
func expandHome(path string) string {
	if runtime.GOOS == "windows" {
		path = expandWindowsEnv(path)
		if strings.HasPrefix(path, `~\`) {
			path = "~/" + path[2:]
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// expandWindowsEnv expands %NAME% references; unknown names are kept.
func expandWindowsEnv(path string) string {
	var out strings.Builder
	for {
		start := strings.IndexByte(path, '%')
		if start < 0 {
			break
		}
		end := strings.IndexByte(path[start+1:], '%')
		if end < 0 {
			break
		}
		name := path[start+1 : start+1+end]
		value, ok := os.LookupEnv(name)
		out.WriteString(path[:start])
		if ok && name != "" {
			out.WriteString(value)
		} else {
			out.WriteString(path[start : start+end+2])
		}
		path = path[start+end+2:]
	}
	out.WriteString(path)
	return out.String()
}
