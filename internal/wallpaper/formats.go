package wallpaper

import (
	"image"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"aether/internal/platform"
)

// IsImageFile reports whether the filename has a supported still-image extension.
func IsImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".jfif", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

// DetectImage sniffs an image file's content and returns its MIME type and
// canonical extension ("jpeg"/"png"/...). It is used when a download URL has no
// file extension and for extensionless files the user picked themselves.
func DetectImage(path string) (mime, ext string, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", false
	}
	defer f.Close()
	_, format, err := image.DecodeConfig(f)
	if err != nil {
		return "", "", false
	}
	switch strings.ToLower(format) {
	case "jpeg":
		return "image/jpeg", ".jpg", true
	case "png":
		return "image/png", ".png", true
	case "gif":
		return "image/gif", ".gif", true
	case "bmp":
		return "image/bmp", ".bmp", true
	case "webp":
		return "image/webp", ".webp", true
	}
	return "", "", false
}

// EnsureImageExtension adds the canonical extension to an extensionless image
// file, so the gallery, extraction, previews and the theme writer recognize it.
// Files that already have a supported extension are returned unchanged.
func EnsureImageExtension(path string) (string, error) {
	if IsImageFile(path) {
		return path, nil
	}
	if filepath.Ext(path) != "" {
		return path, nil
	}
	_, ext, ok := DetectImage(path)
	if !ok {
		return path, nil
	}
	dest := path + ext
	if _, err := os.Stat(dest); err == nil {
		// A previous download already published the named copy.
		return dest, nil
	}
	if err := platform.Rename(path, dest); err != nil {
		return path, err
	}
	return dest, nil
}
