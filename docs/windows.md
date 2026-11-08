# Windows

Aether runs on Windows 10 and 11. It extracts palettes like it does on Linux and applies them to the Windows desktop and to Windows apps: the wallpaper, light/dark mode and accent colour, Windows Terminal, LeopardWM, komorebi, VS Code, Zed, Neovim, Alacritty and WezTerm.

Nothing on your desktop changes until you choose a target and click **Apply Theme**. Every change can be undone with **Settings → Windows → Restore** or `aether --restore-windows`.

## Install

Aether is a single portable executable. Download it, put it in a folder of your choice (for example `%LOCALAPPDATA%\Programs\Aether`), and start it.

| File | Use |
|------|-----|
| `aether.exe` | The app. Double-click it to open the editor. CLI commands also work from a terminal. |
| `aether-cli.exe` | The same program built as a console app, for scripts and pipelines. |

Aether uses the Microsoft Edge WebView2 runtime, which Windows 11 ships with. On Windows 10, install the [Evergreen runtime](https://developer.microsoft.com/microsoft-edge/webview2/) if Aether reports it missing.

### Build from source

On Windows, with Go 1.25+, Node.js 22+ and the Wails CLI:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
git clone https://github.com/omacom/aether.git
cd aether
wails build
```

From Linux or WSL you can cross-compile; no C toolchain is needed:

```bash
make build-windows    # writes build/bin/aether.exe and build/bin/aether-cli.exe
```

`make build-windows` runs `wails build -platform windows/amd64 -skipbindings`, which reuses the checked-in frontend bindings because generating them requires a native build.

## Where Aether keeps its files

| Purpose | Location |
|---------|----------|
| Settings, blueprints, custom templates, generated theme | `%APPDATA%\aether\` |
| Restore record, installed-file list | `%LOCALAPPDATA%\aether\data\` |
| Thumbnails, palette cache, blurred wallpapers (safe to delete) | `%LOCALAPPDATA%\aether\cache\` |
| Wallpaper library and downloaded wallpapers | `%USERPROFILE%\Pictures\Wallpapers` |

Downloads from Wallhaven, GitHub and favorites exports are saved in the wallpaper library, the folder the **Local** tab scans. A downloaded wallpaper therefore shows up in the gallery and in Explorer right away. If you picked another folder under **Settings → Wallpaper folder**, downloads go there instead.

If `XDG_CONFIG_HOME`, `XDG_DATA_HOME` or `XDG_CACHE_HOME` are set, Aether uses them as it does on Linux (Neovim and WezTerm honour them on Windows too).

## Targets

Choose targets in the editor's **Targets** strip or under **App templates**. Each target is an Aether template you can override by copying it into `%APPDATA%\aether\custom\` (see [Custom Templates](custom-templates.md)).

| Target | What Aether changes | Reload |
|--------|---------------------|--------|
| **Windows wallpaper** (`windows-wallpaper.json`) | Sets the theme wallpaper on every monitor, the fit style (`fill` by default) and the desktop colour shown around a fitted image. | Immediate |
| **Windows mode & accent** (`windows.json`) | Sets apps and the shell to light or dark mode and the accent colour (Start, taskbar, title bars, controls) under `HKEY_CURRENT_USER`. Turns off *Automatically pick an accent colour from my background* if it was on, so the accent sticks. Uncomment `accentOnTaskbar` or `accentOnTitleBars` in the template to also show the accent there. | Broadcasts `WM_SETTINGCHANGE`; most apps update at once |
| **Windows Terminal** (`windows-terminal.json`) | Adds or updates the `Aether` colour scheme and window theme in `settings.json` of every installed build (Store, Preview, Canary, unpackaged) and selects them for **Settings → Defaults**, which every profile without its own scheme inherits. Set `"applyTo": "default"` to change only your default profile, or `"none"` to only add the scheme. Comments and other settings are kept. | Windows Terminal reloads `settings.json` itself |
| **LeopardWM** (`leopardwm.toml`) | Sets the `[appearance]` colours in `%APPDATA%\leopardwm\config\config.toml`: `active_border_color` and the tab strip colours. Every other line of the file stays as it is. | Runs `lwm reload` |
| **komorebi** (`komorebi.json`) | Merges a custom Base16 `theme` into `komorebi.json` (in `%KOMOREBI_CONFIG_HOME%` or your profile). Your border role choices, such as `unfocused_border`, are kept. | Runs `komorebic replace-configuration` |
| **VS Code** | Installs the Aether colour theme extension into `%USERPROFILE%\.vscode\extensions`. Pick *Aether* with **Preferences: Color Theme**. | Reload the window |
| **Zed** | Writes `%APPDATA%\Zed\themes\aether.json`. Pick *Aether* in the theme selector. | Next start |
| **Neovim** | Writes `lua\plugins\aether.lua` (a lazy.nvim spec) into your Neovim config (`%LOCALAPPDATA%\nvim`, or `%XDG_CONFIG_HOME%\nvim` when set). | `:Lazy reload aether` or restart |
| **Alacritty** | Writes `%APPDATA%\alacritty\aether.toml`. Add `import = ["~/AppData/Roaming/alacritty/aether.toml"]` under `[general]` in `alacritty.toml` once. | Alacritty reloads imports |
| **WezTerm** | Writes `~\.config\wezterm\aether.lua` (or under `%XDG_CONFIG_HOME%`). Add `config.colors = require("aether")` to your WezTerm config once. | The file asks WezTerm to watch it |

Aether only writes into the configuration of apps you have set up: when `komorebi.json`, LeopardWM's `config.toml`, Windows Terminal's `settings.json` or the Neovim, Alacritty and WezTerm config folders do not exist, the target is skipped.

Linux-only targets (Hyprland, Waybar, Kitty, Ghostty, Foot, Mako, btop, the icon theme and others) are hidden on Windows, and Windows targets are hidden on Linux.

## Undo and backups

Before Aether changes a Windows setting for the first time it records the previous value in `%LOCALAPPDATA%\aether\data\windows-restore.json`, including a copy of your wallpaper. Later themes never overwrite that record.

- **Settings → Windows → Restore**, or `aether --restore-windows`, puts back the wallpaper, desktop colour, light/dark mode, accent colour, the Windows Terminal scheme and theme selection, and the LeopardWM and komorebi colours, then removes the record.
- Files Aether patches keep two backups next to them:
  - `<file>.aether-original.bak`: the file before Aether first changed it (never overwritten),
  - `<file>.aether-previous.bak`: the file before the latest change.

  For Windows Terminal that is `%LOCALAPPDATA%\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json.aether-original.bak`.
- When Aether is about to replace a generated file (for example `aether.toml` for Alacritty) that someone else edited, it keeps that version as `<file>.aether-original.bak`.

To undo by hand, copy an `.aether-original.bak` over the file, and pick your previous accent colour and wallpaper in **Settings → Personalization**.

## Command line

`aether.exe` is a GUI program; when you run a CLI command from a terminal it attaches to that terminal for its output. Because Windows does not wait for GUI programs, the prompt can return before the output finishes. Use `aether-cli.exe` in scripts:

```powershell
aether-cli --extract-palette $HOME\Pictures\Wallpapers\forest.jpg --json
aether-cli --generate .\forest.jpg --no-apply --output $env:TEMP\aether-preview
aether-cli --list-apps
```

`aether --generate <wallpaper>` without `--no-apply` applies **every** Windows target, like it applies every app on Linux. Use the GUI to pick targets, or `--no-apply` to only render the files.

Windows-only commands:

| Command | Purpose |
|---------|---------|
| `aether --restore-windows` | Undo the Windows changes Aether made |
| `aether --register-url-handler` | Open `aether://` links with this Aether |
| `aether --unregister-url-handler` | Remove the `aether://` handler |

### Remote control

The running editor listens on the named pipe `\\.\pipe\aether-<your user SID>` instead of a Unix socket. Only your user account can connect. `aether status`, `aether extract`, `aether apply` and the other [remote control](remote-control.md) commands work unchanged.

## aether:// links

On Linux Aether registers `aether://` itself. On Windows it never does so on its own: click **Register** under **Settings → Windows**, or run `aether --register-url-handler`. This writes `HKEY_CURRENT_USER\Software\Classes\aether` pointing to the current `aether.exe`; register again after you move it. **Remove** or `aether --unregister-url-handler` deletes the key.

Any web page can build an `aether://` link, so on Windows a link with `silent=true` applies only the targets you enabled in the editor; with none enabled it only writes the theme files. Links without `silent=true` open Aether's confirmation dialog as on Linux.

## Custom apps

[Custom apps](custom-apps.md) work on Windows with these differences:

- `destination` may use `%APPDATA%`-style variables and `~\`.
- Creating symlinks needs Developer Mode or an elevated Aether. Without them Aether copies the generated file instead, and only replaces a copy it made itself.
- Post-apply hooks can be `post-apply.ps1` (run with PowerShell), `post-apply.cmd` or `post-apply.bat`. `post-apply.sh` runs when `bash` (Git Bash, MSYS2) is on `PATH`.

## Updates

`aether upgrade`, and the update indicator in the title bar, download `aether-windows-amd64.exe` from the latest GitHub release and replace the running executable. A release without a Windows build is not offered as an update.

## Limitations

- One wallpaper is set for all monitors. Per-monitor wallpapers (`IDesktopWallpaper`) are not supported yet.
- Windows does not repaint every surface on `WM_SETTINGCHANGE`; some title bars and the Start menu pick up a new accent colour after the app restarts or you sign out and in.
- Desktop icon themes and Omarchy features (native themes, shell plugins, the background-only switch through Omarchy) are Linux-only and hidden.
- Templates for apps you run inside WSL are not applied to the WSL distro yet; see below.

## WSL apps (design note, not implemented)

Tools you run inside WSL (Neovim, btop, Kitty under WSLg) read their config from the distro, not from Windows. Two ways to theme them are possible:

1. **Delegate to Linux Aether in the distro.** Windows Aether writes `colors.toml` and the wallpaper to a shared folder and runs `wsl.exe -d <distro> -- aether --import-colors-toml <path>` followed by an apply. This reuses every Linux template, reload hook and the Omarchy integration unchanged, and needs Aether installed in the distro. It is the preferred approach.
2. **Write Linux templates over the WSL share.** Windows Aether renders the Linux-only templates and writes them to `\\wsl.localhost\<distro>\home\<user>\.config\aether\theme\`, then runs reload hooks with `wsl.exe -d <distro> --exec`. This needs no Linux Aether but cannot create Linux symlinks from Windows, so destinations would be copies, and the distro, user and home directory have to be configured (or read with `wsl.exe -l -q` and `wsl.exe -d <distro> -e sh -c 'echo $HOME'`).

Either way the target would be opt-in per distro, use the same restore record, and never touch a distro that is not running unless the user asks.
