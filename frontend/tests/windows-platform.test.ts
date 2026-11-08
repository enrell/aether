import {beforeEach, expect, test, vi} from 'vitest';
import SettingsView from '../src/lib/components/settings/SettingsView.svelte';
import SettingsSidebar from '../src/lib/components/sidebar/SettingsSidebar.svelte';
import {render, settle} from './setup';

const app = vi.hoisted(() => ({
    platform: 'windows',
    GetSettings: vi.fn(),
    SaveSettings: vi.fn(),
    GetURLHandlerStatus: vi.fn(),
    HasWindowsRestore: vi.fn(),
    RegisterURLHandler: vi.fn(),
    UnregisterURLHandler: vi.fn(),
    RestoreWindowsSettings: vi.fn(),
    GetTemplateColors: vi.fn(),
    ListInstalledIconThemes: vi.fn(),
    GetOmarchyCapabilities: vi.fn(),
}));

vi.mock('../wailsjs/go/main/App', () => app);
vi.mock('../src/lib/stores/platform.svelte', () => ({
    getPlatform: () => app.platform,
    getPlatformKnown: () => app.platform !== '',
    getIsWindows: () => app.platform === 'windows',
    initPlatform: vi.fn().mockResolvedValue(undefined),
}));

beforeEach(() => {
    for (const fn of Object.values(app)) {
        if (typeof fn === 'function') vi.mocked(fn).mockReset();
    }
    app.GetSettings.mockResolvedValue({});
    app.GetTemplateColors.mockResolvedValue({});
    app.ListInstalledIconThemes.mockResolvedValue([]);
    app.GetOmarchyCapabilities.mockResolvedValue({
        available: false,
        overrideApps: [],
    });
    app.GetURLHandlerStatus.mockResolvedValue({
        supported: true,
        registered: false,
        current: false,
        command: '',
    });
    app.HasWindowsRestore.mockResolvedValue(true);
    app.RestoreWindowsSettings.mockResolvedValue('Restored 9 Windows settings');
});

async function renderSettings(platform: string) {
    app.platform = platform;
    const view = render(SettingsView, {});
    await settle();
    await settle();
    return view.target;
}

function button(target: HTMLElement, label: string) {
    return [...target.querySelectorAll('button')].find(
        b => b.textContent?.trim() === label
    );
}

test('Windows settings offer opt-in link registration and restore', async () => {
    const target = await renderSettings('windows');
    expect(target.textContent).toContain('aether:// links');
    expect(target.textContent).toContain('Not registered');
    // Nothing is registered until the user asks for it.
    expect(app.RegisterURLHandler).not.toHaveBeenCalled();

    button(target, 'Register')!.click();
    await settle();
    expect(app.RegisterURLHandler).toHaveBeenCalledTimes(1);

    button(target, 'Restore')!.click();
    await settle();
    expect(app.RestoreWindowsSettings).toHaveBeenCalledTimes(1);
});

test('Linux settings have no Windows section', async () => {
    const target = await renderSettings('linux');
    expect(target.textContent).not.toContain('aether:// links');
    expect(app.GetURLHandlerStatus).not.toHaveBeenCalled();
});

test('the icon theme picker is hidden on Windows only', async () => {
    for (const [platform, visible] of [
        ['windows', false],
        ['linux', true],
    ] as const) {
        app.platform = platform;
        const {target, destroy} = render(SettingsSidebar, {});
        await settle();
        await settle();
        const hasIcons =
            target.querySelector('[aria-label^="Choose icon theme"]') !== null;
        expect(hasIcons, platform).toBe(visible);
        await destroy();
    }
});
