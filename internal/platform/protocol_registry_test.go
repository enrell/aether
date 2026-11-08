package platform

import (
	"testing"

	"aether/internal/winapi"
)

func TestWindowsURLHandlerRegistration(t *testing.T) {
	reg := winapi.NewMemoryRegistry()
	exe := `C:\Users\me\Apps\Aether\aether.exe`
	if _, ok, _ := WindowsURLHandler(reg); ok {
		t.Fatal("handler registered before opting in")
	}
	if err := RegisterWindowsURLHandler(reg, exe); err != nil {
		t.Fatal(err)
	}
	command, ok, err := WindowsURLHandler(reg)
	if err != nil || !ok || command != `"C:\Users\me\Apps\Aether\aether.exe" --handle-url "%1"` {
		t.Errorf("command = %q %v %v", command, ok, err)
	}
	if v, ok, _ := reg.Get(`Software\Classes\aether`, "URL Protocol"); !ok || v.Kind != winapi.KindString {
		t.Error(`missing "URL Protocol" marker`)
	}
	if err := UnregisterWindowsURLHandler(reg); err != nil {
		t.Fatal(err)
	}
	if keys := reg.Keys(); len(keys) != 0 {
		t.Errorf("keys left after unregister: %v", keys)
	}
}

func TestUnregisterKeepsForeignHandler(t *testing.T) {
	reg := winapi.NewMemoryRegistry()
	_ = reg.Set(`Software\Classes\aether\shell\open\command`, "", winapi.String(`"C:\other.exe" "%1"`))
	if err := UnregisterWindowsURLHandler(reg); err == nil {
		t.Error("removed another program's handler")
	}
	if _, ok, _ := WindowsURLHandler(reg); !ok {
		t.Error("foreign handler deleted")
	}
}
