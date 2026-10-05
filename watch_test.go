package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestWatcher(t *testing.T) (*fileWatcher, <-chan string) {
	t.Helper()
	changes := make(chan string, 16)
	fw, err := newFileWatcher(func(path string) { changes <- path })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fw.Close() })
	return fw, changes
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// expectChange waits for exactly one change notification for path.
func expectChange(t *testing.T, changes <-chan string, path string) {
	t.Helper()
	select {
	case got := <-changes:
		if got != path {
			t.Fatalf("change for %s, want %s", got, path)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("no change reported for %s", path)
	}
	expectNoChange(t, changes)
}

func expectNoChange(t *testing.T, changes <-chan string) {
	t.Helper()
	select {
	case got := <-changes:
		t.Fatalf("unexpected change for %s", got)
	case <-time.After(3 * reloadDebounce):
	}
}

func TestFileWatcherSaveStyles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "v0")

	fw, changes := newTestWatcher(t)
	if err := fw.Add(path); err != nil {
		t.Fatal(err)
	}

	saves := []struct {
		name string
		save func()
	}{
		{"in place", func() { writeFile(t, path, "v1") }},
		{"atomic rename over original", func() {
			writeFile(t, path+".tmp", "v2")
			if err := os.Rename(path+".tmp", path); err != nil {
				t.Fatal(err)
			}
		}},
		{"in place after atomic", func() { writeFile(t, path, "v3") }},
		{"rename original to backup, write new", func() {
			if err := os.Rename(path, path+"~"); err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, "v4")
			os.Remove(path + "~")
		}},
		{"in place after backup", func() { writeFile(t, path, "v5") }},
	}
	for _, s := range saves {
		t.Run(s.name, func(t *testing.T) {
			s.save()
			expectChange(t, changes, path)
		})
	}
}

func TestFileWatcherIgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "v0")

	fw, changes := newTestWatcher(t)
	if err := fw.Add(path); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "other.md"), "x")
	expectNoChange(t, changes)
}

func TestFileWatcherRemoveIsRefCounted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	writeFile(t, path, "v0")

	fw, changes := newTestWatcher(t)
	fw.Add(path)
	fw.Add(path) // second tab showing the same file

	fw.Remove(path)
	writeFile(t, path, "v1")
	expectChange(t, changes, path)

	fw.Remove(path)
	writeFile(t, path, "v2")
	expectNoChange(t, changes)
}
