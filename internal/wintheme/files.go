package wintheme

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"aether/internal/platform"
)

// Backup suffixes for configuration files Aether patches in place.
const (
	// OriginalBackupSuffix marks the copy taken before Aether first changed
	// the file. It is never overwritten.
	OriginalBackupSuffix = ".aether-original.bak"
	// PreviousBackupSuffix marks the copy taken before the latest change.
	PreviousBackupSuffix = ".aether-previous.bak"
)

// writeConfig atomically replaces a user configuration file after backing up
// its current content. Unchanged content is not written.
func writeConfig(path string, before, after []byte) (changed bool, err error) {
	if bytes.Equal(before, after) {
		return false, nil
	}
	original := path + OriginalBackupSuffix
	if !exists(original) {
		if err := platform.WriteText(original, string(before)); err != nil {
			return false, fmt.Errorf("back up %s: %w", path, err)
		}
	}
	if err := platform.WriteText(path+PreviousBackupSuffix, string(before)); err != nil {
		return false, fmt.Errorf("back up %s: %w", path, err)
	}
	if err := platform.WriteText(path, string(after)); err != nil {
		return false, err
	}
	return true, nil
}

var manifestMu sync.Mutex

// installFile writes a generated file into an application's configuration
// directory. Aether remembers the hash of what it wrote; when the destination
// was changed by someone else since, it is backed up before being replaced.
func (e *Env) installFile(dst string, content []byte) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	manifestPath := filepath.Join(e.StateDir, "installed-files.json")
	manifest := map[string]string{}
	if data, err := os.ReadFile(manifestPath); err == nil {
		_ = json.Unmarshal(data, &manifest)
	}
	key := strings.ToLower(filepath.Clean(dst))

	if current, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(current, content) {
			return nil
		}
		if manifest[key] != hash(current) {
			backup := dst + OriginalBackupSuffix
			if !exists(backup) {
				if err := platform.WriteText(backup, string(current)); err != nil {
					return fmt.Errorf("back up %s: %w", dst, err)
				}
				e.logf("Backed up %s to %s", dst, backup)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := platform.WriteText(dst, string(content)); err != nil {
		return err
	}
	manifest[key] = hash(content)
	return platform.WriteJSON(manifestPath, manifest)
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
