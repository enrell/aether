//go:build windows

package platform

import (
	"os/exec"
	"syscall"
)

// hideCommandWindow keeps helper processes (post-apply hooks, explorers) from
// flashing a console window.
func hideCommandWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
