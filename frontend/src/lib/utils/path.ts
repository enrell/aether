// Path helpers that work with both POSIX paths and Windows drive paths.
// Aether receives absolute paths from Go (filepath.Join on the host OS), so
// the frontend must not assume "/" as the separator.

function isWindowsPath(path: string): boolean {
    return /^[A-Za-z]:[\\/]/.test(path) || path.startsWith('\\\\');
}

// pathKey normalizes a local path for comparison. Windows paths are
// case-insensitive; POSIX paths and URLs are left untouched.
export function pathKey(path: string): string {
    if (!path || path.startsWith('http://') || path.startsWith('https://')) {
        return path;
    }
    return isWindowsPath(path) ? path.toLowerCase() : path;
}

export function samePath(a: string, b: string): boolean {
    return pathKey(a) === pathKey(b);
}

export function includesPath(list: readonly string[], path: string): boolean {
    return list.some(candidate => samePath(candidate, path));
}

export function pathBasename(path: string): string {
    const index = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));
    return index >= 0 ? path.slice(index + 1) : path;
}

export function pathDirname(path: string): string {
    const index = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));
    if (index < 0) return '';
    if (index === 0) return '/';
    return path.slice(0, index);
}

// pathDisplayDir shortens the user's home prefix to "~" on Linux, macOS and
// Windows, so the wallpaper metadata row stays readable.
export function pathDisplayDir(path: string): string {
    const dir = pathDirname(path);
    return dir.replace(/^(?:\/home|\/Users|[A-Za-z]:\\Users)[\\/][^\\/]+/, '~');
}
