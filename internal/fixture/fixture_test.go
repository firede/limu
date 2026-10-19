package fixture

import (
	"encoding/json"
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
	count := 0
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
		if filepath.Base(filepath.Dir(path)) != "pele" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "testdata" || !strings.HasSuffix(path, ".json.gz") {
			return nil
		}
		count++
		data, err := Read(strings.TrimSuffix(path, ".gz"))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		if err := Check(data); err != nil {
			rel, _ := filepath.Rel(top, path)
			t.Errorf("%s: %v", rel, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 80 {
		t.Errorf("%d fixtures, fewer than the 80 the oracle writes", count)
	}
}

func TestCheck(t *testing.T) {
	pinned := `"pele":{"version":"1.6.0","commit":"44da152bbb79b91f03e68016f42499bb25bf6e82"}`
	for _, c := range []struct {
		json string
		ok   bool
	}{
		{`{` + pinned + `,"cases":[1]}`, true},
		{`{` + pinned + `,"cases":[1],"more":{"x":[]}}`, true},
		{`{` + pinned + `,"cases":[]}`, false},
		{`{` + pinned + `,"cases":[ ]}`, false},
		{`{"pele":{"version":"1.6.0","commit":"0000000"},"cases":[1]}`, false},
		{`{"cases":[1]}`, false},
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
