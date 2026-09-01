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

- `main.go` — CLI, IPC (Unix socket), webview, file watching
- `template.html` — embedded via `go:embed`; tab UI, GitHub markdown CSS, zoom/theme JS
- **Single-process tabs:** First invocation starts the viewer; subsequent invocations send file paths via Unix socket (`$XDG_RUNTIME_DIR/md-viewer-UID.sock`). Each file opens as a tab.
- **Detach pattern:** Parent process launches child with `MD_VIEWER_DETACHED=1` env var and `Setsid: true`, then exits. Child opens webview and runs until window closed.
- **Live reload:** `fsnotify` watches opened files; on write, re-renders markdown and pushes to webview via `updateTabContent()` JS call.
- **Rendering pipeline:** Read .md file → goldmark GFM → HTML string → injected into webview via JS `addTab()`
- **Theme/Zoom:** Dark theme default, 180% default zoom. CSS custom properties for theming. `localStorage` wrapped in try/catch (data URI origin restriction).
- **In-tab search:** `/` opens search bar, `Esc` closes it, `F3`/`Shift+F3` navigate matches. Uses DOM TreeWalker to highlight text nodes with `<mark>` elements; highlights are cleared on close or new search.
- **Tab management:** `Ctrl+Tab`/`Ctrl+PgDown` switch to next tab, `Ctrl+Shift+Tab`/`Ctrl+PgUp` to previous. `Ctrl+W` or middle-click closes a tab.

## Constraints

- `webview.SetHtml()` loads pages as data URIs — no `localStorage` origin, so all storage calls must be wrapped in try/catch.
- Dependencies must be pinned to versions compatible with Go 1.22 (e.g. fsnotify v1.7.0, not v1.10+).

## Conventions

- Module path: `md-viewer`
- Binary name: `md-viewer`
- No external CSS/JS; all styling is embedded inline in the HTML template
