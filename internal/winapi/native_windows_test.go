//go:build windows

package winapi

import (
	"fmt"
	"os"
	"testing"
)

// TestNativeRegistry exercises the real HKCU registry under a throwaway key.
// It only runs when AETHER_TEST_REAL_REGISTRY=1 (set in CI) so `go test` on
// a desktop never writes to its registry.
func TestNativeRegistry(t *testing.T) {
	if os.Getenv("AETHER_TEST_REAL_REGISTRY") != "1" {
		t.Skip("set AETHER_TEST_REAL_REGISTRY=1 to write to HKCU\\Software\\AetherTest-*")
	}
	system, err := Native()
	if err != nil {
		t.Fatal(err)
	}
	reg := system.Registry
	key := fmt.Sprintf(`Software\AetherTest-%d`, os.Getpid())
	t.Cleanup(func() { _ = reg.DeleteKey(key) })

	if _, ok, err := reg.Get(key, "missing"); err != nil || ok {
		t.Fatalf("Get on a missing key = %v, %v", ok, err)
	}
	values := map[string]Value{
		"dword":  DWORD(0xc40078d4),
		"string": String("10"),
		"binary": Binary([]byte{0x99, 0xeb, 0xff, 0}),
		"":       String("default value"),
	}
	for name, v := range values {
		if err := reg.Set(key+`\sub`, name, v); err != nil {
			t.Fatal(err)
		}
	}
	for name, want := range values {
		got, ok, err := reg.Get(key+`\sub`, name)
		if err != nil || !ok || !got.Equal(want) {
			t.Errorf("Get(%q) = %+v %v %v, want %+v", name, got, ok, err, want)
		}
	}
	if err := reg.Delete(key+`\sub`, "dword"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := reg.Get(key+`\sub`, "dword"); ok {
		t.Error("value survived Delete")
	}
	if err := reg.DeleteKey(key); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := reg.Get(key+`\sub`, "string"); ok {
		t.Error("subkey survived DeleteKey")
	}
	if err := reg.DeleteKey(key); err != nil {
		t.Errorf("DeleteKey of a missing key: %v", err)
	}
}

// TestNativeWallpaperRead only reads the current wallpaper.
func TestNativeWallpaperRead(t *testing.T) {
	system, err := Native()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.Desktop.Wallpaper(); err != nil {
		t.Errorf("Wallpaper(): %v", err)
	}
}
