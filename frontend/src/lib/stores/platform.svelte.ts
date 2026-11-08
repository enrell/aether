// The operating system Aether runs on, as reported by Go's runtime.GOOS
// ("linux", "darwin" or "windows"). Empty until loaded or outside Wails.
let platform = $state('');
let initialization: Promise<void> | null = null;

export function getPlatform(): string {
    return platform;
}

export function getIsWindows(): boolean {
    return platform === 'windows';
}

export function initPlatform(): Promise<void> {
    initialization ??= (async () => {
        try {
            const {GetPlatform} = await import('../../../wailsjs/go/main/App');
            platform = await GetPlatform();
        } catch {
            platform = '';
        }
    })();
    return initialization;
}
