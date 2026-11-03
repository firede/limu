package fixture

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// FullCases is how many cases of a fixture of drawings hold their lines in full. Past them the
// oracle writes a digest of each case's lines in their place, as FULL in
// internal/oracle/run.mjs does; the two are the same number.
const FullCases = 300

// Digest is the digest the oracle writes of the lines of a drawing: the first 16 bytes of the
// SHA-256 of the lines, each in UTF-8 followed by a newline, in lowercase hexadecimal.
func Digest(lines []string) string {
	h := sha256.New()
	for _, line := range lines {
		h.Write([]byte(line))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// Drawing is what Pele drew of one case of a fixture of drawings, a render.json: the lines
// the oracle read out of Pele's SVG, or, past the fixture's first FullCases cases, a digest of
// them, with the lines the suite keeps in full by their index. A test embeds it in the type
// of its cases.
type Drawing struct {
	Lines  []string       `json:"lines"`
	Digest string         `json:"digest"`
	Kept   map[int]string `json:"kept"`
}

// Line is line i of what Pele drew where the fixture holds it, and "" where it does not.
func (d *Drawing) Line(i int) string {
	if d.Digest == "" {
		if i < len(d.Lines) {
			return d.Lines[i]
		}
		return ""
	}
	return d.Kept[i]
}

// LineRead is Line for the lines a test writes when it reads Pele's with read: line i of what
// Pele drew once the lines read reads as "" are taken out, as Line gives it, where the fixture
// holds it, and "" where it does not.
func (d *Drawing) LineRead(read func(string) string) func(int) string {
	var gone []int
	if d.Digest == "" {
		for k, line := range d.Lines {
			if read(line) == "" {
				gone = append(gone, k)
			}
		}
	} else {
		for k, line := range d.Kept {
			if read(line) == "" {
				gone = append(gone, k)
			}
		}
		slices.Sort(gone)
	}
	return func(i int) string {
		for _, k := range gone {
			if k <= i {
				i++
			}
		}
		return d.Line(i)
	}
}

// matches says whether got, the lines limu wrote, are the lines of the digest, read with read:
// the lines kept in full read as got holds them, and the lines in their places and the rest
// of got together those of the digest.
func (d *Drawing) matches(got []string, read func(string) string) bool {
	lines, k, _ := d.restore(got, read)
	return k < 0 && Digest(lines) == d.Digest
}

// restore is Pele's lines as got holds them: got with each line kept in full in its place,
// where read reads it as got holds it there, or put in before what got holds there, where read
// reads it as "", a line limu writes nothing for. Where got does not hold a kept line as read
// reads it, restore gives that line's index in Pele's lines and in got, and otherwise -1.
func (d *Drawing) restore(got []string, read func(string) string) ([]string, int, int) {
	keys := make([]int, 0, len(d.Kept))
	for k := range d.Kept {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	lines := make([]string, 0, len(got)+len(keys))
	g := 0
	for _, k := range keys {
		for len(lines) < k && g < len(got) {
			lines = append(lines, got[g])
			g++
		}
		line := d.Kept[k]
		if r := read(line); r != "" {
			if len(lines) < k || g >= len(got) || got[g] != r {
				return nil, k, g
			}
			g++
		}
		lines = append(lines, line)
	}
	return append(lines, got[g:]...), -1, -1
}

// Drawings is a fixture of drawings, as LoadDrawings reads it, for comparing what limu draws of
// its cases with what Pele drew.
type Drawings struct {
	path string
	once sync.Once
	full []Drawing
	err  error
}

// LoadDrawings reads the fixture of drawings at path into v, as Load does.
func LoadDrawings(path string, v any) (*Drawings, error) {
	if err := Load(path, v); err != nil {
		return nil, err
	}
	return &Drawings{path: path}, nil
}

// fullPath is where ./scripts/oracle.sh --full writes the lines of every case of the fixture,
// beside it: render.full.json for render.json, kept as render.full.json.gz.
func (f *Drawings) fullPath() string {
	return strings.TrimSuffix(f.path, ".json") + ".full.json"
}

// suite is the oracle's suite that writes the fixture: x-render writes render.json in the
// package x, whose directory the test runs in.
func (f *Drawings) suite() string {
	dir, err := os.Getwd()
	if err != nil {
		return "<suite>"
	}
	return filepath.Base(dir) + "-" + strings.TrimSuffix(filepath.Base(f.path), ".json")
}

// fullCases reads the cases of the file of full lines once.
func (f *Drawings) fullCases() ([]Drawing, error) {
	f.once.Do(func() {
		var v struct {
			Cases []Drawing `json:"cases"`
		}
		f.err = Load(f.fullPath(), &v)
		f.full = v.Cases
	})
	return f.full, f.err
}

// Diff says how got, the lines limu wrote of case i of the fixture, differ from d, what Pele
// drew of it, or is "" where they do not. A test that reads some of Pele's lines through
// limu's own code, as the author's style or the arcs of a path, gives read, which is applied
// to each of Pele's lines before they are compared; the suite keeps in full each line read
// changes, as the line cannot be found again from what limu wrote. read gives "" for a line
// limu writes nothing in place of, as a series color limu leaves unset. read may be nil.
//
// Where the fixture holds the lines, Diff names the first that differs. Where it holds only
// a digest, Diff takes the lines from the file ./scripts/oracle.sh --full writes beside the
// fixture, where they are those of the digest, and otherwise says how to write that file.
func (f *Drawings) Diff(i int, d *Drawing, got []string, read func(string) string) string {
	if read == nil {
		read = func(line string) string { return line }
	}
	if d.Digest == "" {
		return lineDiff(got, readAll(d.Lines, read), "")
	}
	if d.matches(got, read) {
		return ""
	}
	full, err := f.fullCases()
	write := fmt.Sprintf("./scripts/oracle.sh --full <pele> %s", f.suite())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		msg := fmt.Sprintf("differs from what Pele drew, of which %s holds only a digest; %s writes every line to %s.gz, to show the first that differs", f.path, write, f.fullPath())
		if _, k, g := d.restore(got, read); k >= 0 {
			msg += fmt.Sprintf(". Of the lines it holds, this one differs: line %d\n got %s\nPele %s", g, At(got, g), read(d.Kept[k]))
		}
		return msg
	case err != nil:
		return fmt.Sprintf("differs from what Pele drew, of which %s holds only a digest, and %s.gz cannot be read: %v", f.path, f.fullPath(), err)
	case i >= len(full) || Digest(full[i].Lines) != d.Digest:
		return fmt.Sprintf("differs from what Pele drew, of which %s holds only a digest, and %s.gz does not hold the lines of that digest: it is stale, and %s writes it again", f.path, f.fullPath(), write)
	}
	if msg := lineDiff(got, readAll(full[i].Lines, read), fmt.Sprintf("\n(Pele's lines from %s.gz)", f.fullPath())); msg != "" {
		return msg
	}
	return fmt.Sprintf("differs from the digest of what Pele drew in %s, but not from its lines in %s.gz: read changes a line the suite does not keep in full", f.path, f.fullPath())
}

// readAll is lines, each read with read, without those read reads as "".
func readAll(lines []string, read func(string) string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if r := read(line); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// lineDiff names the first line where got and want differ, with both, and note after them, or
// is "" where they are the same.
func lineDiff(got, want []string, note string) string {
	i := FirstDiff(got, want)
	if i < 0 {
		return ""
	}
	return fmt.Sprintf("line %d\n got %s\nPele %s%s", i, At(got, i), At(want, i), note)
}

// At is line i of lines, or "(none)" past their end.
func At(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(none)"
}
