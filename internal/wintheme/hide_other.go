//go:build !windows

package wintheme

import "os/exec"

func hideWindow(*exec.Cmd) {}
