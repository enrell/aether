//go:build windows

package winapi

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	user32                    = windows.NewLazySystemDLL("user32.dll")
	procSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	procSendMessageTimeoutW   = user32.NewProc("SendMessageTimeoutW")
	procSetSysColors          = user32.NewProc("SetSysColors")
)

const (
	spiGetDeskWallpaper = 0x0073
	spiSetDeskWallpaper = 0x0014
	spifUpdateIniFile   = 0x01
	spifSendChange      = 0x02

	hwndBroadcast    = 0xffff
	wmSettingChange  = 0x001a
	smtoAbortIfHung  = 0x0002
	broadcastTimeout = 5000 // milliseconds

	colorDesktop = 1 // COLOR_DESKTOP / COLOR_BACKGROUND
)

// Native returns the real HKCU registry and desktop APIs.
func Native() (System, error) {
	return System{Registry: nativeRegistry{}, Desktop: nativeDesktop{}}, nil
}

type nativeRegistry struct{}

func (nativeRegistry) Get(key, name string) (Value, bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return Value{}, false, nil
	}
	if err != nil {
		return Value{}, false, fmt.Errorf(`open HKCU\%s: %w`, key, err)
	}
	defer k.Close()

	_, kind, err := k.GetValue(name, nil)
	if errors.Is(err, registry.ErrNotExist) {
		return Value{}, false, nil
	}
	if err != nil {
		return Value{}, false, fmt.Errorf(`read HKCU\%s\%s: %w`, key, name, err)
	}
	switch kind {
	case registry.DWORD:
		v, _, err := k.GetIntegerValue(name)
		return DWORD(uint32(v)), err == nil, err
	case registry.SZ, registry.EXPAND_SZ:
		v, _, err := k.GetStringValue(name)
		return Value{Kind: Kind(kind), String: v}, err == nil, err
	case registry.BINARY:
		v, _, err := k.GetBinaryValue(name)
		return Binary(v), err == nil, err
	default:
		return Value{}, false, fmt.Errorf(`HKCU\%s\%s has unsupported type %d`, key, name, kind)
	}
}

func (nativeRegistry) Set(key, name string, value Value) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf(`open HKCU\%s: %w`, key, err)
	}
	defer k.Close()
	switch value.Kind {
	case KindDWORD:
		err = k.SetDWordValue(name, value.DWORD)
	case KindString:
		err = k.SetStringValue(name, value.String)
	case KindExpandString:
		err = k.SetExpandStringValue(name, value.String)
	case KindBinary:
		err = k.SetBinaryValue(name, value.Binary)
	default:
		err = fmt.Errorf("unsupported registry type %s", value.Kind)
	}
	if err != nil {
		return fmt.Errorf(`write HKCU\%s\%s: %w`, key, name, err)
	}
	return nil
}

func (nativeRegistry) Delete(key, name string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(`open HKCU\%s: %w`, key, err)
	}
	defer k.Close()
	if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf(`delete HKCU\%s\%s: %w`, key, name, err)
	}
	return nil
}

func (r nativeRegistry) DeleteKey(key string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.ENUMERATE_SUB_KEYS)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(`open HKCU\%s: %w`, key, err)
	}
	subkeys, err := k.ReadSubKeyNames(-1)
	k.Close()
	if err != nil {
		return fmt.Errorf(`list HKCU\%s: %w`, key, err)
	}
	for _, sub := range subkeys {
		if err := r.DeleteKey(key + `\` + sub); err != nil {
			return err
		}
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf(`delete HKCU\%s: %w`, key, err)
	}
	return nil
}

type nativeDesktop struct{}

func (nativeDesktop) Wallpaper() (string, error) {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	ok, _, err := procSystemParametersInfoW.Call(
		spiGetDeskWallpaper,
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
		0,
	)
	if ok == 0 {
		return "", fmt.Errorf("SystemParametersInfo(SPI_GETDESKWALLPAPER): %w", err)
	}
	return windows.UTF16ToString(buf), nil
}

func (nativeDesktop) SetWallpaper(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	ok, _, callErr := procSystemParametersInfoW.Call(
		spiSetDeskWallpaper,
		0,
		uintptr(unsafe.Pointer(p)),
		spifUpdateIniFile|spifSendChange,
	)
	if ok == 0 {
		return fmt.Errorf("SystemParametersInfo(SPI_SETDESKWALLPAPER): %w", callErr)
	}
	return nil
}

func (nativeDesktop) SetDesktopColor(r, g, b uint8) error {
	element := int32(colorDesktop)
	colorref := uint32(r) | uint32(g)<<8 | uint32(b)<<16
	ok, _, err := procSetSysColors.Call(1, uintptr(unsafe.Pointer(&element)), uintptr(unsafe.Pointer(&colorref)))
	if ok == 0 {
		return fmt.Errorf("SetSysColors: %w", err)
	}
	return nil
}

func (nativeDesktop) BroadcastSettingChange(area string) error {
	p, err := windows.UTF16PtrFromString(area)
	if err != nil {
		return err
	}
	var result uintptr
	ok, _, callErr := procSendMessageTimeoutW.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(p)),
		smtoAbortIfHung,
		broadcastTimeout,
		uintptr(unsafe.Pointer(&result)),
	)
	// A timeout from a hung window is not fatal; the setting is already stored.
	if ok == 0 && !errors.Is(callErr, syscall.Errno(0)) && !errors.Is(callErr, windows.ERROR_TIMEOUT) {
		return fmt.Errorf("broadcast WM_SETTINGCHANGE(%s): %w", strings.TrimSpace(area), callErr)
	}
	return nil
}
