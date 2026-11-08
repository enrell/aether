package platform

import (
	"os"
	"runtime"
	"strings"
)

// ExpandUserPath expands a leading "~/" or "~\" and, on Windows, %VARIABLE%
// references such as %APPDATA%. Unknown variables and paths without a home
// prefix are returned unchanged.
func ExpandUserPath(path string) string {
	if runtime.GOOS == "windows" {
		path = expandWindowsEnv(path)
		if strings.HasPrefix(path, `~\`) {
			path = "~/" + path[2:]
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}

func expandWindowsEnv(path string) string {
	var out strings.Builder
	for {
		start := strings.IndexByte(path, '%')
		if start < 0 {
			break
		}
		end := strings.IndexByte(path[start+1:], '%')
		if end < 0 {
			break
		}
		name := path[start+1 : start+1+end]
		value, ok := os.LookupEnv(name)
		out.WriteString(path[:start])
		if ok && name != "" {
			out.WriteString(value)
		} else {
			out.WriteString(path[start : start+end+2])
		}
		path = path[start+end+2:]
	}
	out.WriteString(path)
	return out.String()
}
