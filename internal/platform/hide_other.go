//go:build !windows

package platform

import "os/exec"

func hideCommandWindow(*exec.Cmd) {}
