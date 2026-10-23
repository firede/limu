package fixture

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// root is the directory of the module, the nearest one above that holds go.mod.
func root(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test")
		}
		dir = parent
	}
}

// Every fixture of the module was written from the pinned version of Pele and holds cases.
func TestFixturesArePinned(t *testing.T) {
	top := root(t)
	count, drawings := 0, 0
	err := filepath.WalkDir(top, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == top {
				return nil
			}
			// Hidden directories hold worktrees and tools' files, and a directory with a go.mod
			// of its own is another module.
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		// The full lines of a fixture of drawings are not tracked, and are not a fixture.
		if filepath.Base(filepath.Dir(path)) != "pele" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "testdata" ||
			!strings.HasSuffix(path, ".json.gz") || strings.HasSuffix(path, ".full.json.gz") {
			return nil
		}
		count++
		data, err := Read(strings.TrimSuffix(path, ".gz"))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		rel, _ := filepath.Rel(top, path)
		if err := Check(data); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
		if filepath.Base(path) == "render.json.gz" {
			drawings++
			if err := checkDrawings(data); err != nil {
				t.Errorf("%s: %v", rel, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 80 {
		t.Errorf("%d fixtures, fewer than the 80 the oracle writes", count)
	}
	if drawings < 33 {
		t.Errorf("%d fixtures of drawings, fewer than the 33 the oracle writes", drawings)
	}
}

// checkDrawings says what is wrong with the shape of a fixture of drawings, if anything: its
// first FullCases cases hold their lines, and each of the rest a digest of them in their
// place, with the lines it keeps in full. A case where Pele drew nothing, as one it refused
// with an error, holds neither.
func checkDrawings(data []byte) error {
	var f struct {
		Cases []map[string]json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	digests := 0
	for i, c := range f.Cases {
		var d Drawing
		raw, _ := json.Marshal(c)
		if err := json.Unmarshal(raw, &d); err != nil {
			return fmt.Errorf("case %d: %v", i, err)
		}
		_, lines := c["lines"]
		_, digest := c["digest"]
		switch {
		case i < FullCases && (digest || d.Kept != nil):
			return fmt.Errorf("case %d, of the first %d, holds a digest in place of its lines", i, FullCases)
		case i >= FullCases && lines:
			return fmt.Errorf("case %d, past the first %d, holds its lines in place of a digest", i, FullCases)
		case digest && !isDigest(d.Digest):
			return fmt.Errorf("case %d holds %q, not a digest", i, d.Digest)
		case !digest && d.Kept != nil:
			return fmt.Errorf("case %d keeps lines beside no digest", i)
		}
		if digest {
			digests++
		}
	}
	if digests == 0 {
		return fmt.Errorf("no digests in %d cases", len(f.Cases))
	}
	return nil
}

// isDigest says whether s is written as Digest writes one: 32 lowercase hexadecimal digits.
func isDigest(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, r := range s {
		if !('0' <= r && r <= '9' || 'a' <= r && r <= 'f') {
			return false
		}
	}
	return true
}

func TestCheck(t *testing.T) {
	pinned := `"pele":{"version":"1.9.0","commit":"e7f092a4de147cc32210913c20e36eaa5fcd0ef1","arch":"arm64","node":"v24.22.0"}`
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{` + pinned + `,"cases":[1]}`, true},
		{`{` + pinned + `,"cases":[1],"more":{"x":[]}}`, true},
		{`{` + pinned + `,"cases":[]}`, false},
		{`{` + pinned + `,"cases":[ ]}`, false},
		{`{"pele":{"version":"1.9.0","commit":"0000000"},"cases":[1]}`, false},
		{`{"pele":{"version":"1.8.0","commit":"8fad23bd51783e4df1204e177a7fb0d2a30dcd13"},"cases":[1]}`, false},
		{`{"cases":[1]}`, false},
		// The architecture and the version of Node are recorded and not compared.
		{`{"pele":{"version":"1.9.0","commit":"e7f092a4de147cc32210913c20e36eaa5fcd0ef1"},"cases":[1]}`, true},
		{`{"pele":{"version":"1.9.0","commit":"e7f092a4de147cc32210913c20e36eaa5fcd0ef1","arch":"x64","node":"v26.0.0"},"cases":[1]}`, true},
	} {
		if err := Check([]byte(c.json)); (err == nil) != c.ok {
			t.Errorf("Check(%s) = %v", c.json, err)
		}
	}
}

func TestNumbers(t *testing.T) {
	for _, v := range []float64{0, math.Copysign(0, -1), 1.5, -2, math.NaN(), math.Inf(1), math.Inf(-1), 1e300} {
		raw, err := json.Marshal(Number(v))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Float(raw)
		if err != nil {
			t.Fatal(err)
		}
		if math.Float64bits(got) != math.Float64bits(v) && !(math.IsNaN(got) && math.IsNaN(v)) {
			t.Errorf("Float(%s) = %v, not %v", raw, got, v)
		}
	}
	if _, err := Float(json.RawMessage(`{"$n":"x"}`)); err == nil {
		t.Error(`Float({"$n":"x"}) is no error`)
	}
}
