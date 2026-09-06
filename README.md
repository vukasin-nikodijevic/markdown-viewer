# md-viewer

A lightweight CLI tool that previews Markdown files in a native webview window with GitHub Flavored Markdown rendering.

## Features

- GitHub Flavored Markdown (tables, task lists, strikethrough, autolinks)
- Native webview window (GTK/WebKit on Linux)
- **Tabbed interface** — open multiple files in one window: pass several files at once, or invoke repeatedly
- **Live reload** — file changes are re-rendered automatically on save
- **Dark / Light theme** — dark by default, toggle with toolbar button or `Ctrl+Shift+L`
- **Zoom** — 180% default, adjust with `Ctrl+` / `Ctrl-` / `Ctrl+0`
- **In-tab search** — press `/` or `Ctrl+F` to search, `F3`/`Shift+F3` to navigate matches
- **Quick index** — press `Ctrl+Shift+A` for a live index of all tabs & headings; type to filter, `Enter`/click to jump
- **Tab management** — switch tabs with `Ctrl+Tab`/`Ctrl+PgDown`, close with `Ctrl+W` or middle-click
- Background process — CLI returns immediately, process exits when window closes

## Requirements

- Go 1.22+
- Linux: `libgtk-3-dev` and `libwebkit2gtk-4.1-dev`

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
```

## Build & Install

```bash
make
sudo make install        # installs to /usr/local/bin
# or
PREFIX=~/.local make install  # user-local install
```

## Usage

```bash
# Open a file (starts viewer in background)
md-viewer README.md

# Open several files at once — each becomes its own tab
md-viewer README.md AGENTS.md go.mod

# Open another file as a new tab in the existing viewer
md-viewer Makefile
```

## Keyboard Shortcuts

| Shortcut           | Action             |
| ------------------ | ------------------ |
| `Ctrl+=`           | Zoom in            |
| `Ctrl+-`           | Zoom out           |
| `Ctrl+0`           | Reset zoom         |
| `Ctrl+Shift+L`     | Toggle theme       |
| `/`                | Open search        |
| `Ctrl+F`           | Open search        |
| `Ctrl+Shift+A`     | Quick index (all tabs & headings) |
| `Esc`              | Close search / index |
| `F3`               | Next match         |
| `Shift+F3`         | Previous match     |
| `Ctrl+Tab`         | Next tab           |
| `Ctrl+PgDown`      | Next tab           |
| `Ctrl+Shift+Tab`   | Previous tab       |
| `Ctrl+PgUp`        | Previous tab       |
| `Ctrl+W`           | Close tab          |
| Middle-click tab   | Close tab          |
