# md-viewer

A lightweight CLI tool that previews Markdown files in a native webview window with GitHub Flavored Markdown rendering.

## Features

- GitHub Flavored Markdown (tables, task lists, strikethrough, autolinks)
- Native webview window (GTK/WebKit on Linux)
- Background process — CLI returns immediately, process exits when window closes

## Requirements

- Go 1.22+
- Linux: `libgtk-3-dev` and `libwebkit2gtk-4.0-dev`

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev
```

## Build

```bash
go build -o md-viewer .
```

## Usage

```bash
./md-viewer README.md
```

## Example Table

| Feature | Supported |
|---------|-----------|
| Tables | ✅ |
| Task lists | ✅ |
| Strikethrough | ✅ |
| Autolinks | ✅ |
| Headings | ✅ |
| Code blocks | ✅ |

## Task List

- [x] Markdown parsing
- [x] GFM support
- [x] Webview rendering
- [x] Background process
- [ ] File watching (live reload)
