package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

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

// trySendToExisting connects to a running viewer and sends file paths
// (one per line on a single connection).
func trySendToExisting(sockPath string, paths []string) bool {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return false
	}
	defer conn.Close()
	for _, p := range paths {
		if _, err := fmt.Fprintln(conn, p); err != nil {
			return false
		}
	}
	return true
}

type openTab struct {
	id   int64
	path string
}

// buildPage renders the template with the given tabs injected.
func buildPage(tabs []openTab) string {
	var init strings.Builder
	init.WriteString("initTabs([")
	for i, t := range tabs {
		if i > 0 {
			init.WriteString(",")
		}
		body, err := renderMarkdown(t.path)
		if err != nil {
			body = "<p><strong>Error:</strong> " + html.EscapeString(err.Error()) + "</p>"
		}
		titleJSON, _ := json.Marshal(filepath.Base(t.path))
		htmlJSON, _ := json.Marshal(body)
		fmt.Fprintf(&init, "[%d,%s,%s]", t.id, titleJSON, htmlJSON)
	}
	init.WriteString("]);")
	return strings.Replace(templateHTML, "/* INITIAL_TAB */", init.String(), 1)
}

// servePage serves page() on a random loopback port behind a random token
// and returns the URL to navigate to.
func servePage(page func() string) (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		ln.Close()
		return "", err
	}
	host := ln.Addr().String()
	pagePath := "/" + hex.EncodeToString(tokenBytes) + "/"
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Host check guards against DNS rebinding from other web pages.
		if r.Host != host || r.URL.Path != pagePath {
			http.NotFound(rw, r)
			return
		}
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Header().Set("Cache-Control", "no-store")
		io.WriteString(rw, page())
	})
	go http.Serve(ln, handler)
	return "http://" + host + pagePath, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: md-viewer <file.md> [file.md ...]\n")
		os.Exit(1)
	}

	absPaths := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		absPath, err := filepath.Abs(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stat(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		absPaths = append(absPaths, absPath)
	}

	sockPath := socketPath()

	if os.Getenv(envDetached) != "1" {
		if trySendToExisting(sockPath, absPaths) {
			for _, p := range absPaths {
				fmt.Printf("Opened %s in existing viewer\n", filepath.Base(p))
			}
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

		cmd := exec.Command(exe, absPaths...)
		cmd.Env = append(os.Environ(), envDetached+"=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		cmd.Stdin = nil
		cmd.Stdout = devNull
		cmd.Stderr = devNull

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting viewer: %v\n", err)
			os.Exit(1)
		}

		for _, p := range absPaths {
			fmt.Printf("Previewing %s (pid: %d)\n", filepath.Base(p), cmd.Process.Pid)
		}
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

	w := webview.New(false)
	defer w.Destroy()
	setupMenu()
	w.SetTitle("MD Preview")
	w.SetSize(960, 720, webview.HintNone)

	var tabID atomic.Int64
	// Open tabs in display order; the page is rebuilt from this on (re)load.
	var mu sync.Mutex
	var tabs []openTab

	reloadFile := func(path string) {
		body, err := renderMarkdown(path)
		if err != nil {
			return
		}
		var ids []int64
		mu.Lock()
		for _, t := range tabs {
			if t.path == path {
				ids = append(ids, t.id)
			}
		}
		mu.Unlock()
		htmlJSON, _ := json.Marshal(body)
		for _, id := range ids {
			js := fmt.Sprintf("updateTabContent(%d, %s);", id, htmlJSON)
			w.Dispatch(func() { w.Eval(js) })
		}
	}

	// Live reload: re-render a file's tabs whenever it is saved.
	watcher, err := newFileWatcher(reloadFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	addTabForFile := func(path string) {
		body, err := renderMarkdown(path)
		if err != nil {
			return
		}
		id := tabID.Add(1)
		mu.Lock()
		tabs = append(tabs, openTab{id: id, path: path})
		mu.Unlock()
		watcher.Add(path)

		titleJSON, _ := json.Marshal(filepath.Base(path))
		htmlJSON, _ := json.Marshal(body)
		js := fmt.Sprintf("addTab(%d, %s, %s);", id, titleJSON, htmlJSON)
		w.Dispatch(func() { w.Eval(js) })
	}

	w.Bind("updateTitle", func(title string) {
		w.Dispatch(func() { w.SetTitle("MD Preview — " + title) })
	})
	w.Bind("closeViewer", func() {
		w.Dispatch(func() { w.Terminate() })
	})
	w.Bind("tabClosed", func(id int64) {
		mu.Lock()
		defer mu.Unlock()
		for i, t := range tabs {
			if t.id == id {
				tabs = append(tabs[:i], tabs[i+1:]...)
				watcher.Remove(t.path)
				return
			}
		}
	})

	// Ctrl/Cmd+O: native open dialog, starting in the active tab's folder.
	w.Bind("openFileDialog", func(activeID int64) {
		dir := ""
		mu.Lock()
		for _, t := range tabs {
			if t.id == activeID {
				dir = filepath.Dir(t.path)
			}
		}
		mu.Unlock()
		// Run the modal from the main loop, not inside the JS message callback.
		w.Dispatch(func() {
			for _, p := range pickFiles(dir) {
				addTabForFile(p)
			}
		})
	})

	for _, p := range absPaths {
		tabs = append(tabs, openTab{id: tabID.Add(1), path: p})
		watcher.Add(p)
	}

	// Serve the page over loopback HTTP instead of SetHtml: a page loaded
	// from an HTML string has no URL, so "Reload" shows a blank page (macOS).
	// Each load re-renders the currently open tabs from disk.
	pageURL, err := servePage(func() string {
		mu.Lock()
		snapshot := append([]openTab(nil), tabs...)
		mu.Unlock()
		return buildPage(snapshot)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting page server: %v\n", err)
		os.Exit(1)
	}
	w.Navigate(pageURL)

	// Accept new files from other invocations
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			scanner := bufio.NewScanner(conn)
			for scanner.Scan() {
				addTabForFile(scanner.Text())
			}
			conn.Close()
		}
	}()

	w.Run()
}
