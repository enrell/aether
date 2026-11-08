// The operating system Aether runs on, as reported by Go's runtime.GOOS
// ("linux", "darwin" or "windows"). Empty until loaded or outside Wails.
let platform = $state('');
let initialization: Promise<void> | null = null;

export function getPlatform(): string {
    return platform;
}

// getPlatformKnown reports whether the platform lookup has completed. Gates
// that hide Linux-only UI use this so a slow or failed lookup does not flash
// the Linux interface on Windows.
export function getPlatformKnown(): boolean {
    return platform !== '';
}

export function getIsWindows(): boolean {
    return platform === 'windows';
}

export function initPlatform(): Promise<void> {
    if (platform !== '') return Promise.resolve();
    if (initialization) return initialization;
    initialization = (async () => {
        try {
            const {GetPlatform} = await import('../../../wailsjs/go/main/App');
            platform = await GetPlatform();
        } catch {
            platform = '';
        } finally {
            // A failed lookup may be retried on the next call. A successful
            // one leaves `platform` set, so the early return above wins.
            initialization = null;
        }
    })();
    return initialization;
}
