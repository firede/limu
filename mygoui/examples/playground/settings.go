package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// settings are what the playground remembers between runs: the text last edited and the
// appearance chosen. They are kept in a small JSON file, and an error reading or writing
// it is ignored: the playground then starts as it does the first time.
type settings struct {
	Source     string `json:"source"`
	Appearance string `json:"appearance"`
}

// settingsFile is where the settings are kept, or "" when the system has no place for
// them.
func settingsFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "limu-playground", "settings.json")
}

// loadSettings reads the settings from path, and reports whether there were any.
func loadSettings(path string) (settings, bool) {
	var s settings
	if path == "" {
		return s, false
	}
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &s) != nil {
		return settings{}, false
	}
	return s, true
}

// A settingsWriter writes the settings from a goroutine of its own, so that the window
// does not wait for the disk, keeping only the newest when they change faster than they
// are written.
type settingsWriter struct {
	path    string
	mu      sync.Mutex
	pending *settings
	writing bool
	wg      sync.WaitGroup
}

// save writes s, after any write under way.
func (w *settingsWriter) save(s settings) {
	if w == nil || w.path == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = &s
	if w.writing {
		return
	}
	w.writing = true
	w.wg.Go(w.run)
}

func (w *settingsWriter) run() {
	for {
		w.mu.Lock()
		s := w.pending
		w.pending = nil
		if s == nil {
			w.writing = false
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
		writeSettings(w.path, *s)
	}
}

// wait returns once the settings saved are written.
func (w *settingsWriter) wait() {
	if w != nil {
		w.wg.Wait()
	}
}

// writeSettings writes s to path, through a file beside it, so that a write cut short
// leaves the settings as they were. Errors are ignored.
func writeSettings(path string, s settings) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) != nil {
		return
	}
	if os.Rename(tmp, path) != nil {
		os.Remove(tmp)
	}
}
