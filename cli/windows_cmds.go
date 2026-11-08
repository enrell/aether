package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"aether/internal/platform"
	"aether/internal/theme"
	"aether/internal/winapi"
)

func requireWindows(cmd string) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s is only available on Windows\n", cmd)
	return false
}

// runRegisterURLHandler registers aether:// for the current Windows user.
func runRegisterURLHandler(args []string) int {
	if !requireWindows("--register-url-handler") {
		return 1
	}
	system, err := winapi.Native()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: locate Aether: %v\n", err)
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if previous, ok, _ := platform.WindowsURLHandler(system.Registry); ok && previous != platform.URLHandlerCommand(exe) {
		fmt.Printf("Replacing the previous aether:// handler: %s\n", previous)
	}
	if err := platform.RegisterWindowsURLHandler(system.Registry, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("aether:// links now open %s\n", exe)
	fmt.Println("Remove the handler with: aether --unregister-url-handler")
	return 0
}

func runUnregisterURLHandler(args []string) int {
	if !requireWindows("--unregister-url-handler") {
		return 1
	}
	system, err := winapi.Native()
	if err == nil {
		err = platform.UnregisterWindowsURLHandler(system.Registry)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Println("Removed the aether:// handler.")
	return 0
}

// runRestoreWindows puts back the Windows settings Aether changed.
func runRestoreWindows(args []string) int {
	if !requireWindows("--restore-windows") {
		return 1
	}
	result, err := theme.WindowsEnv().Restore()
	if result.Registry > 0 {
		fmt.Printf("Restored %d registry values (mode, accent colour, wallpaper style)\n", result.Registry)
	}
	if result.Wallpaper != "" {
		fmt.Printf("Restored wallpaper: %s\n", result.Wallpaper)
	}
	for _, f := range result.Files {
		fmt.Printf("Restored settings in %s\n", f)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
