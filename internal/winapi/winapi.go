// Package winapi wraps the few Windows APIs Aether uses to theme the desktop:
// HKCU registry values, the desktop wallpaper and colour, and the
// WM_SETTINGCHANGE broadcast. Callers depend on the Registry and Desktop
// interfaces so the theming logic can be tested with the in-memory fakes in
// this package on any platform.
package winapi

import (
	"errors"
	"fmt"
)

// ErrUnsupported is returned by Native on platforms other than Windows.
var ErrUnsupported = errors.New("windows desktop APIs are not available on this platform")

// Kind is a registry value type. The numbers match the Win32 REG_* constants.
type Kind uint32

const (
	KindString       Kind = 1 // REG_SZ
	KindExpandString Kind = 2 // REG_EXPAND_SZ
	KindBinary       Kind = 3 // REG_BINARY
	KindDWORD        Kind = 4 // REG_DWORD
)

func (k Kind) String() string {
	switch k {
	case KindString:
		return "REG_SZ"
	case KindExpandString:
		return "REG_EXPAND_SZ"
	case KindBinary:
		return "REG_BINARY"
	case KindDWORD:
		return "REG_DWORD"
	default:
		return fmt.Sprintf("REG_%d", uint32(k))
	}
}

// Value is a typed registry value. Only the field matching Kind is used.
type Value struct {
	Kind   Kind   `json:"kind"`
	String string `json:"string,omitempty"`
	Binary []byte `json:"binary,omitempty"`
	DWORD  uint32 `json:"dword,omitempty"`
}

// DWORD returns a REG_DWORD value.
func DWORD(v uint32) Value { return Value{Kind: KindDWORD, DWORD: v} }

// String returns a REG_SZ value.
func String(v string) Value { return Value{Kind: KindString, String: v} }

// Binary returns a REG_BINARY value.
func Binary(v []byte) Value {
	return Value{Kind: KindBinary, Binary: append([]byte(nil), v...)}
}

// Equal reports whether two values have the same type and data.
func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case KindDWORD:
		return v.DWORD == other.DWORD
	case KindBinary:
		return string(v.Binary) == string(other.Binary)
	default:
		return v.String == other.String
	}
}

// Registry reads and writes values below HKEY_CURRENT_USER. Key paths use
// backslashes and are relative to HKCU, e.g.
// `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`. The empty
// value name addresses the key's default value.
type Registry interface {
	// Get returns ok=false when the key or the value does not exist.
	Get(key, name string) (value Value, ok bool, err error)
	// Set creates the key when needed.
	Set(key, name string, value Value) error
	// Delete removes a value. A missing key or value is not an error.
	Delete(key, name string) error
	// DeleteKey removes a key and its subkeys. A missing key is not an error.
	DeleteKey(key string) error
}

// Desktop changes session-wide desktop state.
type Desktop interface {
	// Wallpaper returns the current wallpaper path ("" when none is set).
	Wallpaper() (string, error)
	// SetWallpaper sets the wallpaper on every monitor and persists it.
	SetWallpaper(path string) error
	// SetDesktopColor sets the solid colour shown behind or instead of the
	// wallpaper for the current session.
	SetDesktopColor(r, g, b uint8) error
	// BroadcastSettingChange sends WM_SETTINGCHANGE to all top-level windows
	// with area as the lParam string (e.g. "ImmersiveColorSet").
	BroadcastSettingChange(area string) error
}

// System bundles the APIs a Windows theming target may use.
type System struct {
	Registry Registry
	Desktop  Desktop
}
