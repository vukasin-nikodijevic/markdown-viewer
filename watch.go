package main

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// reloadDebounce coalesces the burst of events a single save produces
// (truncate + write, or write tmp + rename) and avoids rendering a
// half-written file.
const reloadDebounce = 100 * time.Millisecond

// fileWatcher reports changes to a set of files on Linux (inotify) and
// macOS (kqueue). It watches each file's parent directory rather than the
// file itself: editors that save atomically (write a temp file and rename
// it over the original, or rename the original to a backup first) replace
// the inode, which silently ends a per-file watch.
type fileWatcher struct {
	w        *fsnotify.Watcher
	onChange func(path string)

	mu     sync.Mutex
	files  map[string]int // watched file → number of Add calls
	dirs   map[string]int // watched dir → number of watched files in it
	timers map[string]*time.Timer
}

func newFileWatcher(onChange func(path string)) (*fileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &fileWatcher{
		w:        w,
		onChange: onChange,
		files:    make(map[string]int),
		dirs:     make(map[string]int),
		timers:   make(map[string]*time.Timer),
	}
	go fw.run()
	return fw, nil
}

// Add starts watching path. Calls are reference counted, so the same file
// may be added once per tab showing it.
func (fw *fileWatcher) Add(path string) error {
	path = filepath.Clean(path)
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.files[path]++
	if fw.files[path] > 1 {
		return nil
	}
	dir := filepath.Dir(path)
	fw.dirs[dir]++
	if fw.dirs[dir] > 1 {
		return nil
	}
	if err := fw.w.Add(dir); err != nil {
		fw.files[path]--
		if fw.files[path] == 0 {
			delete(fw.files, path)
		}
		fw.dirs[dir]--
		if fw.dirs[dir] == 0 {
			delete(fw.dirs, dir)
		}
		return err
	}
	return nil
}

// Remove undoes one Add call for path.
func (fw *fileWatcher) Remove(path string) {
	path = filepath.Clean(path)
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.files[path] == 0 {
		return
	}
	fw.files[path]--
	if fw.files[path] > 0 {
		return
	}
	delete(fw.files, path)
	if t := fw.timers[path]; t != nil {
		t.Stop()
		delete(fw.timers, path)
	}
	dir := filepath.Dir(path)
	fw.dirs[dir]--
	if fw.dirs[dir] == 0 {
		delete(fw.dirs, dir)
		fw.w.Remove(dir)
	}
}

func (fw *fileWatcher) Close() error {
	return fw.w.Close()
}

func (fw *fileWatcher) run() {
	for {
		select {
		case event, ok := <-fw.w.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				fw.schedule(filepath.Clean(event.Name))
			}
		case _, ok := <-fw.w.Errors:
			if !ok {
				return
			}
		}
	}
}

func (fw *fileWatcher) schedule(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.files[path] == 0 {
		return // another file in a watched directory
	}
	if t := fw.timers[path]; t != nil {
		t.Reset(reloadDebounce)
		return
	}
	fw.timers[path] = time.AfterFunc(reloadDebounce, func() {
		fw.mu.Lock()
		delete(fw.timers, path)
		fw.mu.Unlock()
		fw.onChange(path)
	})
}
