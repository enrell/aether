//go:build windows

package update

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"syscall"
)

const createNewConsole = 0x00000010

// openWindowsUpgradeConsole runs `aether upgrade` in a new console window
// that stays open so the user can read the result.
func openWindowsUpgradeConsole(executable string) error {
	cmdExe, err := exec.LookPath("cmd.exe")
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdExe)
	// start /wait makes cmd wait for the GUI-subsystem binary, which attaches
	// to this console for its output.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       fmt.Sprintf(`cmd.exe /k start "" /b /wait "%s" upgrade`, filepath.Clean(executable)),
		CreationFlags: createNewConsole,
	}
	return cmd.Start()
}
