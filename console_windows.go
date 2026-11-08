//go:build windows

package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows"
)

var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

const attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS

// attachConsole sends CLI output to the terminal that started Aether.
// Release builds are GUI programs without a console of their own, so
// `aether --help` would otherwise print nothing. Redirected handles (pipes,
// files) are kept. aether-cli.exe is a console build that needs none of this.
func attachConsole() {
	stdin, stdout, stderr := validStdHandle(windows.STD_INPUT_HANDLE), validStdHandle(windows.STD_OUTPUT_HANDLE), validStdHandle(windows.STD_ERROR_HANDLE)
	if stdout && stderr {
		return
	}
	if r, _, _ := procAttachConsole.Call(uintptr(attachParentProcess)); r == 0 {
		return // started from Explorer or a browser: there is no console
	}
	if !stdout {
		if f := openConsole("CONOUT$", windows.STD_OUTPUT_HANDLE); f != nil {
			os.Stdout = f
		}
	}
	if !stderr {
		if f := openConsole("CONOUT$", windows.STD_ERROR_HANDLE); f != nil {
			os.Stderr = f
		}
	}
	if !stdin {
		if f := openConsole("CONIN$", windows.STD_INPUT_HANDLE); f != nil {
			os.Stdin = f
		}
	}
	log.SetOutput(os.Stderr)
}

func validStdHandle(std uint32) bool {
	h, err := windows.GetStdHandle(std)
	if err != nil || h == 0 || h == windows.InvalidHandle {
		return false
	}
	t, err := windows.GetFileType(h)
	return err == nil && t != windows.FILE_TYPE_UNKNOWN
}

func openConsole(name string, std uint32) *os.File {
	path, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil
	}
	h, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil
	}
	_ = windows.SetStdHandle(std, h)
	return os.NewFile(uintptr(h), name)
}
