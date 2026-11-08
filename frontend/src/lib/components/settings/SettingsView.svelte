<script lang="ts">
    import {onMount} from 'svelte';
    import {getSettings, updateSettings} from '$lib/stores/settings.svelte';
    import {showToast} from '$lib/stores/ui.svelte';
    import {getIsWindows, initPlatform} from '$lib/stores/platform.svelte';

    type URLHandlerStatus = {
        supported: boolean;
        registered: boolean;
        current: boolean;
        command: string;
    };

    let choosingFolder = $state(false);
    let isWindows = $derived(getIsWindows());
    let wallpaperFolder = $derived(
        getSettings().wallpaperFolder ||
            (isWindows ? '%USERPROFILE%\\Pictures\\Wallpapers' : '~/Wallpapers')
    );
    let urlHandler = $state<URLHandlerStatus | null>(null);
    let canRestore = $state(false);
    let busy = $state('');

    let urlHandlerText = $derived(
        !urlHandler?.registered
            ? 'Not registered. Links from the browser will not open Aether.'
            : urlHandler.current
              ? 'Registered for this copy of Aether.'
              : `Registered for another program: ${urlHandler.command}`
    );

    async function refreshWindows() {
        await initPlatform();
        if (!getIsWindows()) return;
        try {
            const {GetURLHandlerStatus, HasWindowsRestore} = await import(
                '../../../../wailsjs/go/main/App'
            );
            urlHandler = (await GetURLHandlerStatus()) as URLHandlerStatus;
            canRestore = await HasWindowsRestore();
        } catch (err) {
            console.error('Windows settings status failed', err);
        }
    }

    async function toggleURLHandler() {
        busy = 'url';
        try {
            const {RegisterURLHandler, UnregisterURLHandler} = await import(
                '../../../../wailsjs/go/main/App'
            );
            if (urlHandler?.current) {
                await UnregisterURLHandler();
                showToast('aether:// links no longer open Aether');
            } else {
                await RegisterURLHandler();
                showToast('aether:// links now open Aether');
            }
        } catch (err) {
            console.error('URL handler change failed', err);
            showToast(`Could not change the aether:// handler: ${err}`, 6000);
        } finally {
            busy = '';
            await refreshWindows();
        }
    }

    async function restoreWindows() {
        busy = 'restore';
        try {
            const {RestoreWindowsSettings} = await import(
                '../../../../wailsjs/go/main/App'
            );
            showToast(await RestoreWindowsSettings(), 6000);
        } catch (err) {
            console.error('RestoreWindowsSettings failed', err);
            showToast(`Restore incomplete: ${err}`, 8000);
        } finally {
            busy = '';
            await refreshWindows();
        }
    }

    onMount(() => {
        void refreshWindows();
    });

    async function chooseWallpaperFolder() {
        choosingFolder = true;
        try {
            const {ChooseWallpaperFolder} = await import(
                '../../../../wailsjs/go/main/App'
            );
            const folder = await ChooseWallpaperFolder();
            if (!folder) return;

            updateSettings({wallpaperFolder: folder});
            showToast('Wallpaper folder updated');
        } catch (err) {
            console.error('ChooseWallpaperFolder failed', err);
            showToast('Failed to set wallpaper folder');
        } finally {
            choosingFolder = false;
        }
    }
</script>

<div class="h-full overflow-y-auto">
    <div class="mx-auto max-w-[760px] px-8 py-11">
        <header class="mb-9">
            <p
                class="text-accent mb-2 text-[10px] font-semibold uppercase tracking-[0.18em]"
            >
                Preferences
            </p>
            <h1
                class="text-fg-primary text-[26px] font-semibold tracking-[-0.01em]"
            >
                Settings
            </h1>
            <p class="text-fg-secondary mt-2 text-[13px] leading-relaxed">
                Configure where Aether finds local wallpapers{isWindows
                    ? ' and how it integrates with Windows'
                    : ''}.
            </p>
        </header>

        <section aria-labelledby="wallpaper-library-heading">
            <h2
                id="wallpaper-library-heading"
                class="text-fg-dimmed mb-2.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
            >
                Wallpaper library
            </h2>

            <div
                class="border-border bg-bg-secondary grid grid-cols-[auto_minmax(0,1fr)] items-start gap-4 border p-[18px] sm:grid-cols-[auto_minmax(0,1fr)_auto]"
            >
                <div
                    class="bg-accent-muted text-accent flex h-9 w-9 shrink-0 items-center justify-center"
                >
                    <svg
                        class="h-4 w-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                    >
                        <path
                            d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"
                        ></path>
                    </svg>
                </div>

                <div class="min-w-0">
                    <div class="text-fg-primary text-[13px] font-semibold">
                        Wallpaper folder
                    </div>
                    <p class="text-fg-dimmed mb-3 mt-[3px] text-[12px]">
                        The Local page scans this folder and all of its
                        subfolders for images.
                    </p>
                    <div
                        class="bg-bg-primary border-border text-fg-secondary truncate border px-[11px] py-[9px] font-mono text-[12px] font-medium"
                        title={wallpaperFolder}
                    >
                        {wallpaperFolder}
                    </div>
                </div>

                <button
                    type="button"
                    class="bg-accent text-accent-fg hover:bg-accent-hover col-span-2 h-8 w-full shrink-0 px-3.5 text-[12px] font-semibold transition-colors disabled:cursor-wait disabled:opacity-60 sm:col-span-1 sm:w-auto"
                    onclick={chooseWallpaperFolder}
                    disabled={choosingFolder}
                >
                    {choosingFolder ? 'Choosing…' : 'Choose folder…'}
                </button>
            </div>
        </section>

        {#if isWindows}
            <section aria-labelledby="windows-heading" class="mt-9">
                <h2
                    id="windows-heading"
                    class="text-fg-dimmed mb-2.5 text-[10px] font-semibold uppercase tracking-[0.14em]"
                >
                    Windows
                </h2>

                <div class="border-border bg-bg-secondary border">
                    <div
                        class="border-border flex flex-col gap-3 border-b p-[18px] sm:flex-row sm:items-start sm:justify-between"
                    >
                        <div class="min-w-0">
                            <div
                                class="text-fg-primary text-[13px] font-semibold"
                            >
                                aether:// links
                            </div>
                            <p class="text-fg-dimmed mt-[3px] text-[12px]">
                                Open theme links from the browser in Aether.
                                Writes a per-user entry under
                                HKEY_CURRENT_USER\Software\Classes.
                            </p>
                            <p
                                class="text-fg-secondary mt-2 break-all text-[12px]"
                            >
                                {urlHandlerText}
                            </p>
                        </div>
                        <button
                            type="button"
                            class="border-border text-fg-primary hover:bg-bg-hover h-8 shrink-0 border px-3.5 text-[12px] font-semibold transition-colors disabled:cursor-wait disabled:opacity-60"
                            onclick={toggleURLHandler}
                            disabled={busy !== '' || !urlHandler?.supported}
                        >
                            {urlHandler?.current ? 'Remove' : 'Register'}
                        </button>
                    </div>

                    <div
                        class="flex flex-col gap-3 p-[18px] sm:flex-row sm:items-start sm:justify-between"
                    >
                        <div class="min-w-0">
                            <div
                                class="text-fg-primary text-[13px] font-semibold"
                            >
                                Restore Windows settings
                            </div>
                            <p class="text-fg-dimmed mt-[3px] text-[12px]">
                                Put back the wallpaper, light/dark mode, accent
                                colour, Windows Terminal scheme and window
                                manager colours you had before Aether first
                                changed them.
                            </p>
                        </div>
                        <button
                            type="button"
                            class="border-border text-fg-primary hover:bg-bg-hover h-8 shrink-0 border px-3.5 text-[12px] font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
                            onclick={restoreWindows}
                            disabled={busy !== '' || !canRestore}
                            title={canRestore
                                ? 'Undo the Windows changes Aether made'
                                : 'Aether has not changed Windows settings'}
                        >
                            Restore
                        </button>
                    </div>
                </div>
            </section>
        {/if}
    </div>
</div>
