package wallpaper

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writePNGAt(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDetectImageFromContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "download")
	writePNGAt(t, path)
	mime, ext, ok := DetectImage(path)
	if !ok || mime != "image/png" || ext != ".png" {
		t.Fatalf("DetectImage = %q, %q, %v", mime, ext, ok)
	}
}

func TestEnsureImageExtensionRenamesExtensionlessDownload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "8f14e45fceea167a")
	writePNGAt(t, path)
	got, err := EnsureImageExtension(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := path + ".png"; got != want {
		t.Fatalf("EnsureImageExtension = %q, want %q", got, want)
	}
	if !IsImageFile(got) {
		t.Errorf("%q is still not recognized as an image", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureImageExtensionKeepsNamedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wall.jpg")
	writePNGAt(t, path) // content is PNG, but the name already has an extension
	got, err := EnsureImageExtension(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("EnsureImageExtension changed %q to %q", path, got)
	}
}
