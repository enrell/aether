//go:build windows

package wintheme

import (
	"os/exec"
	"syscall"
)

// hideWindow keeps console helpers such as komorebic from flashing a window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
