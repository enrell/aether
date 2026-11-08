//go:build !windows

package winapi

// Native returns ErrUnsupported outside Windows.
func Native() (System, error) {
	return System{}, ErrUnsupported
}
