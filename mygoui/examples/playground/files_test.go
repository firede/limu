package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSource(t *testing.T) {
	dir := t.TempDir()
	cases := []struct{ name, content, want string }{
		{"a.mmd", "\uFEFFflowchart LR\n  A --> B\n", "flowchart LR\n  A --> B\n"},
		{"a.txt", "pie\n  \"a\": 1\n", "pie\n  \"a\": 1\n"},
		{"a.md", "# Notes\n\n```js\nx\n```\n\n```mermaid\npie\n  \"a\": 1\n```\nmore\n", "pie\n  \"a\": 1\n"},
		{"b.MD", "\uFEFF~~~~ mermaid\ngitGraph\n  commit\n~~~~\n", "gitGraph\n  commit\n"},
		{"c.md", "flowchart LR\n  A\n", "flowchart LR\n  A\n"},
	}
	for _, tc := range cases {
		path := filepath.Join(dir, tc.name)
		if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := readSource(path)
		if err != nil || got != tc.want {
			t.Errorf("readSource(%s) = %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
	if _, err := readSource(filepath.Join(dir, "none.mmd")); err == nil {
		t.Error("no error for a file that is not there")
	}
}

// What writeSource writes, readSource reads back.
func TestWriteSource(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mmd", "a.md"} {
		path := filepath.Join(dir, name)
		if err := writeSource(path, "flowchart LR\n  A --> B"); err != nil {
			t.Fatal(err)
		}
		got, err := readSource(path)
		if err != nil || got != "flowchart LR\n  A --> B\n" {
			t.Errorf("%s: read back %q, %v", name, got, err)
		}
	}
}

func TestSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "limu-playground", "settings.json")
	if _, ok := loadSettings(path); ok {
		t.Error("settings before any were saved")
	}
	w := &settingsWriter{path: path}
	for i := range 20 {
		w.save(settings{Source: string(rune('a' + i)), Appearance: appearanceDark})
	}
	w.wait()
	s, ok := loadSettings(path)
	if !ok || s.Source != "t" || s.Appearance != appearanceDark {
		t.Errorf("loaded %+v, %v", s, ok)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadSettings(path); ok {
		t.Error("settings read from a broken file")
	}
	// A writer with no place to write does nothing.
	(&settingsWriter{}).save(settings{})
	var none *settingsWriter
	none.save(settings{})
	none.wait()
}
