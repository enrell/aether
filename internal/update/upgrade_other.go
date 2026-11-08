//go:build !windows

package update

import "errors"

func openWindowsUpgradeConsole(string) error {
	return errors.New("not running on Windows")
}
