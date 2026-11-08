package winapi

import (
	"runtime"
	"testing"
)

func TestMemoryRegistryIsCaseInsensitive(t *testing.T) {
	reg := NewMemoryRegistry()
	if err := reg.Set(`Software\Test`, "Value", DWORD(1)); err != nil {
		t.Fatal(err)
	}
	if v, ok, _ := reg.Get(`software\TEST`, "value"); !ok || v.DWORD != 1 {
		t.Errorf("Get = %+v %v", v, ok)
	}
	_ = reg.Set(`Software\Test\Sub`, "", String("x"))
	if err := reg.DeleteKey(`SOFTWARE\test`); err != nil {
		t.Fatal(err)
	}
	if len(reg.Keys()) != 0 {
		t.Errorf("keys = %v", reg.Keys())
	}
}

func TestValueEqual(t *testing.T) {
	if !Binary([]byte{1, 2}).Equal(Binary([]byte{1, 2})) || DWORD(1).Equal(String("1")) || String("a").Equal(String("b")) {
		t.Error("Value.Equal is wrong")
	}
}

func TestNativeUnsupportedOffWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Native is available on Windows")
	}
	if _, err := Native(); err != ErrUnsupported {
		t.Errorf("Native() error = %v, want ErrUnsupported", err)
	}
}
