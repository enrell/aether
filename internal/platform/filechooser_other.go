//go:build !linux

package platform

import (
	"context"
	"errors"
)

// PortalOpenFile is only available on Linux, where xdg-desktop-portal
// provides the user's file chooser. Callers fall back to the toolkit dialog.
func PortalOpenFile(context.Context, FileChooserOptions) (string, error) {
	return "", errors.ErrUnsupported
}
