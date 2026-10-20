# AGENTS.md

## Project Overview

`md-viewer` — A Go CLI tool that renders Markdown files in a native webview window using GitHub Flavored Markdown styling.

## Tech Stack

- **Language:** Go 1.22+
- **Markdown parser:** github.com/yuin/goldmark with GFM extension
- **File watching:** github.com/fsnotify/fsnotify v1.7.0 (live reload on save)
- **Webview:** github.com/webview/webview_go (CGO, requires gtk3 + webkit2gtk-4.0)
- **OS support:** Linux (webkit2gtk), macOS (WebKit), Windows (WebView2)

## Build Requirements (Linux)

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
make
```

- `webview_go` requires `webkit2gtk-4.0` via pkg-config; `.pkg-config/webkit2gtk-4.0.pc` shims it to 4.1.
- The `Makefile` sets `PKG_CONFIG_PATH` automatically.

## Architecture

- `main.go` — CLI, IPC (Unix socket), loopback page server, webview
- `watch.go` — `fileWatcher`: live-reload file watching (tests in `watch_test.go`)
- `template.html` — embedded via `go:embed`; tab UI, GitHub markdown CSS, zoom/theme JS
- **Single-process tabs:** First invocation starts the viewer; subsequent invocations send file paths via Unix socket (`$XDG_RUNTIME_DIR/md-viewer-UID.sock`). Each file opens as a tab.
- **Detach pattern:** Parent process launches child with `MD_VIEWER_DETACHED=1` env var and `Setsid: true`, then exits. Child opens webview and runs until window closed.
- **Live reload:** `fileWatcher` (fsnotify: inotify on Linux, kqueue on macOS) watches each open file's *parent directory* and filters events to open files — a per-file watch is lost when an editor saves atomically (temp file + rename, or backup rename). Watches are ref-counted per tab and dropped when the last tab for a file/dir closes. Write/Create events are debounced (100 ms), then the file is re-rendered and pushed via `updateTabContent()`; an open search is re-applied.
- **Page serving:** The page is served from a loopback HTTP server (`127.0.0.1:<random port>/<random token>/`, Host header checked) and loaded with `w.Navigate()`, not `SetHtml()` — a page loaded from an HTML string has no URL, so right-click → Reload produced a blank page on macOS. Go keeps the ordered list of open tabs (JS reports closes via the `tabClosed` binding); every page load re-renders them from disk. Active tab, zoom and scroll are saved to `sessionStorage` on `beforeunload` and restored after `initTabs()`.
- **Rendering pipeline:** Read .md file → goldmark GFM → HTML string → injected into the page via `initTabs()` on load, or JS `addTab()` for tabs opened later
- **Theme/Zoom:** Dark theme default, 180% default zoom. CSS custom properties for theming. `localStorage` wrapped in try/catch (data URI origin restriction).
- **In-tab search:** `/`, `Ctrl+F` or `Cmd+F` opens search bar, `Esc` closes it, `Enter`/`F3` (with `Shift` for previous) navigate matches. Uses DOM TreeWalker to highlight text nodes with `<mark>` elements; highlights are cleared on close or new search.
- **Quick index:** `Ctrl+Shift+A` opens a popup with a live-filtering index of all tabs and their headings (rebuilt on open). Multi-token filter scored by heading match + tab name match; `↑`/`↓`/Enter or click switches to the item's tab and scrolls to the heading (headings without goldmark IDs get fallback IDs; `scroll-margin-top` keeps them below the tab bar).
- **Help dialog:** `?` toolbar button or `Ctrl+?` toggles a modal listing all shortcuts (`Esc`/click outside closes). The shortcut list is static HTML in `template.html` — keep it in sync when adding shortcuts.
- **macOS menu:** `menu_darwin.go` (cgo/Cocoa) installs an app + Edit menu after `webview.New` — webview creates none, and Cocoa only dispatches Cmd+C/V/X/A/Z via menu key equivalents. Don't add menu items whose key equivalents clash with JS shortcuts (e.g. Cmd+F, Cmd+W). `Ctrl+C` is mapped to `execCommand('copy')` in JS on macOS.
- **Tab management:** `Ctrl+Tab`/`Ctrl+PgDown` switch to next tab, `Ctrl+Shift+Tab`/`Ctrl+PgUp` to previous. `Ctrl+W` or middle-click closes a tab. `Ctrl/Cmd+O` calls the `openFileDialog` binding, which shows a native multi-select picker (`dialog_darwin.go` NSOpenPanel, `dialog_linux.go` GTK) starting in the active tab's folder and opens each file via `addTabForFile`.

## CI / Releases

- `.github/workflows/build.yml` runs `make` + `go test` on native runners (CGO, no cross-compiling): Linux amd64/arm64 on Ubuntu 22.04 (older glibc for compatibility), macOS arm64/amd64.
- CI builds with current stable Go, not go.mod's 1.22: Go 1.22 + current Apple `ld` produces macOS binaries with an invalid code signature (SIGKILL on launch). A smoke-test step runs each binary (and `codesign --verify` on macOS arm64; Intel binaries are unsigned).
- Pushing a `v*` tag also creates a GitHub release with `md-viewer_<version>_<os>_<arch>.tar.gz` archives and `checksums.txt`.
- The Homebrew formula lives in `vukasin-nikodijevic/homebrew-tap` and builds from the tag's source tarball; bump its `url`/`sha256` after tagging.

## Constraints

- Don't switch back to `webview.SetHtml()` — it breaks Reload. The page origin uses a random port per run, so `localStorage` doesn't persist across runs; keep all storage calls wrapped in try/catch.
- Dependencies must be pinned to versions compatible with Go 1.22 (e.g. fsnotify v1.7.0, not v1.10+).

## Conventions

- Module path: `md-viewer`
- Binary name: `md-viewer`
- No external CSS/JS; all styling is embedded inline in the HTML template
