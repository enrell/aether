package platform

import (
	"fmt"
	"strings"

	"aether/internal/winapi"
)

// The aether:// handler on Windows is a per-user URL protocol class under
// HKCU\Software\Classes. Unlike the Linux desktop entry it is never
// registered automatically: users opt in with `aether --register-url-handler`
// (or the button in Settings) and remove it with --unregister-url-handler.

const urlProtocolKey = `Software\Classes\aether`

// URLHandlerCommand is the command line Windows runs for an aether:// link.
func URLHandlerCommand(exe string) string {
	return fmt.Sprintf(`"%s" --handle-url "%%1"`, exe)
}

// RegisterWindowsURLHandler makes exe the current user's aether:// handler.
func RegisterWindowsURLHandler(reg winapi.Registry, exe string) error {
	for _, v := range []struct {
		key, name, value string
	}{
		{urlProtocolKey, "", "URL:Aether theme link"},
		{urlProtocolKey, "URL Protocol", ""},
		{urlProtocolKey + `\DefaultIcon`, "", fmt.Sprintf(`"%s",0`, exe)},
		{urlProtocolKey + `\shell\open\command`, "", URLHandlerCommand(exe)},
	} {
		if err := reg.Set(v.key, v.name, winapi.String(v.value)); err != nil {
			return fmt.Errorf("register aether:// handler: %w", err)
		}
	}
	return nil
}

// WindowsURLHandler returns the registered aether:// command line.
func WindowsURLHandler(reg winapi.Registry) (string, bool, error) {
	v, ok, err := reg.Get(urlProtocolKey+`\shell\open\command`, "")
	if err != nil || !ok {
		return "", false, err
	}
	return v.String, true, nil
}

// UnregisterWindowsURLHandler removes the aether:// handler when it points
// to an Aether executable. A handler registered by another program is left
// alone.
func UnregisterWindowsURLHandler(reg winapi.Registry) error {
	command, ok, err := WindowsURLHandler(reg)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if !strings.Contains(command, "--handle-url") {
		return fmt.Errorf("aether:// is handled by another program (%s); not removing it", command)
	}
	return reg.DeleteKey(urlProtocolKey)
}
