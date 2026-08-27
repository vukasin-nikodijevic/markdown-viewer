package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	webview "github.com/webview/webview_go"
)

//go:embed template.html
var templateHTML string

const envDetached = "MD_VIEWER_DETACHED"

func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, fmt.Sprintf("md-viewer-%d.sock", os.Getuid()))
}

func renderMarkdown(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// trySendToExisting connects to a running viewer and sends the file path.
func trySendToExisting(sockPath, absPath string) bool {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return false
	}
	defer conn.Close()
	_, err = fmt.Fprintln(conn, absPath)
	return err == nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: md-viewer <file.md>\n")
		os.Exit(1)
	}

	absPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sockPath := socketPath()

	if os.Getenv(envDetached) != "1" {
		if trySendToExisting(sockPath, absPath) {
			fmt.Printf("Opened %s in existing viewer\n", filepath.Base(absPath))
			return
		}

		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer devNull.Close()

		cmd := exec.Command(exe, absPath)
		cmd.Env = append(os.Environ(), envDetached+"=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		cmd.Stdin = nil
		cmd.Stdout = devNull
		cmd.Stderr = devNull

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting viewer: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Previewing %s (pid: %d)\n", filepath.Base(absPath), cmd.Process.Pid)
		return
	}

	// --- Detached viewer process ---

	os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating socket: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		ln.Close()
		os.Remove(sockPath)
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("MD Preview")
	w.SetSize(960, 720, webview.HintNone)

	var tabID atomic.Int64
	// Track file path → tab IDs for live reload
	var mu sync.Mutex
	fileToTabs := make(map[string][]int64)

	addTabForFile := func(path string) {
		body, err := renderMarkdown(path)
		if err != nil {
			return
		}
		id := tabID.Add(1)
		title := filepath.Base(path)
		titleJSON, _ := json.Marshal(title)
		htmlJSON, _ := json.Marshal(body)
		js := fmt.Sprintf("addTab(%d, %s, %s);", id, titleJSON, htmlJSON)
		w.Dispatch(func() { w.Eval(js) })

		mu.Lock()
		fileToTabs[path] = append(fileToTabs[path], id)
		mu.Unlock()
		watcher.Add(path)
	}

	reloadFile := func(path string) {
		body, err := renderMarkdown(path)
		if err != nil {
			return
		}
		mu.Lock()
		ids := fileToTabs[path]
		mu.Unlock()
		htmlJSON, _ := json.Marshal(body)
		for _, id := range ids {
			js := fmt.Sprintf("updateTabContent(%d, %s);", id, htmlJSON)
			w.Dispatch(func() { w.Eval(js) })
		}
	}

	w.Bind("updateTitle", func(title string) {
		w.Dispatch(func() { w.SetTitle("MD Preview — " + title) })
	})
	w.Bind("closeViewer", func() {
		w.Dispatch(func() { w.Terminate() })
	})

	// Watch for file changes
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					reloadFile(event.Name)
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	// Inject initial tab into the template before loading
	body, err := renderMarkdown(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering markdown: %v\n", err)
		os.Exit(1)
	}
	titleJSON, _ := json.Marshal(filepath.Base(absPath))
	htmlJSON, _ := json.Marshal(body)
	initScript := fmt.Sprintf("addTab(1, %s, %s);", titleJSON, htmlJSON)
	tabID.Store(1)
	mu.Lock()
	fileToTabs[absPath] = []int64{1}
	mu.Unlock()
	watcher.Add(absPath)
	page := strings.Replace(templateHTML, "/* INITIAL_TAB */", initScript, 1)
	w.SetHtml(page)

	// Accept new files from other invocations
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			scanner := bufio.NewScanner(conn)
			if scanner.Scan() {
				addTabForFile(scanner.Text())
			}
			conn.Close()
		}
	}()

	w.Run()
}
