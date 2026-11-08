package blueprint

import (
	"strings"
)

// safeFileStem turns a user- or URL-supplied blueprint name into a single
// portable file name stem. Both "/" and "\" are path separators on some
// platform, and Windows rejects the remaining characters, so a name must never
// be passed to filepath.Join unsanitized.
func safeFileStem(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*', 0:
			return '-'
		}
		if r < 32 {
			return '-'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	name = strings.Trim(name, " .")
	if strings.Trim(name, " .-") == "" {
		return ""
	}
	switch strings.ToUpper(strings.TrimSuffix(name, ".json")) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}
