// Package fixture reads the fixtures internal/oracle writes from Pele: JSON, gzipped, beside
// where the tests name them, and the numbers in them as the oracle writes them.
package fixture

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

// Origin is the version of Pele a fixture was written from, as the oracle records it, with
// the architecture and the version of Node that ran it.
type Origin struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Arch    string `json:"arch"`
	Node    string `json:"node"`
}

// Pinned is the version of Pele limu follows; every fixture is written from it. Its Arch and
// Node are not compared, as the oracle runs only on arm64 and a fixture holds what Pele makes
// of its cases whichever Node runs it.
var Pinned = Origin{Version: "1.6.0", Commit: "44da152bbb79b91f03e68016f42499bb25bf6e82"}

// Read is the JSON of the fixture at path, which is kept as path + ".gz".
func Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path + ".gz")
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// Check says what is wrong with the JSON of a fixture, if anything: that it was written from
// another version of Pele than the pinned one, or that a list at its top level is empty, as
// one would be if the suite that writes it found no cases.
func Check(data []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return err
	}
	var origin Origin
	if err := json.Unmarshal(top["pele"], &origin); err != nil {
		return fmt.Errorf("no origin: %v", err)
	}
	if origin.Version != Pinned.Version || origin.Commit != Pinned.Commit {
		return fmt.Errorf("written from Pele %s, commit %s, not %s, commit %s", origin.Version, origin.Commit, Pinned.Version, Pinned.Commit)
	}
	for key, raw := range top {
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 && raw[0] == '[' && len(bytes.TrimSpace(raw[1:])) == 1 {
			return fmt.Errorf("%s holds no cases", key)
		}
	}
	return nil
}

// Load reads the fixture at path into v, after Check finds nothing wrong with it.
func Load(path string, v any) error {
	data, err := Read(path)
	if err != nil {
		return err
	}
	if err := Check(data); err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	return json.Unmarshal(data, v)
}

// Number is a number as the oracle's enc writes it: as itself, or as {"$n": …} where JSON
// has no word for it, NaN, the infinities and negative zero.
func Number(v float64) any {
	switch {
	case math.IsNaN(v):
		return map[string]any{"$n": "NaN"}
	case math.IsInf(v, 1):
		return map[string]any{"$n": "Infinity"}
	case math.IsInf(v, -1):
		return map[string]any{"$n": "-Infinity"}
	case v == 0 && math.Signbit(v):
		return map[string]any{"$n": "-0"}
	}
	return v
}

// Float reads a number as the oracle's enc writes it.
func Float(raw json.RawMessage) (float64, error) {
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f, nil
	}
	var tagged struct {
		N string `json:"$n"`
	}
	if err := json.Unmarshal(raw, &tagged); err != nil {
		return 0, err
	}
	switch tagged.N {
	case "NaN":
		return math.NaN(), nil
	case "Infinity":
		return math.Inf(1), nil
	case "-Infinity":
		return math.Inf(-1), nil
	case "-0":
		return math.Copysign(0, -1), nil
	}
	return 0, fmt.Errorf("not a number: %s", raw)
}

// FirstDiff is the index of the first line where a and b differ, or -1 where they are the
// same.
func FirstDiff(a, b []string) int {
	for i := range max(len(a), len(b)) {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			return i
		}
	}
	return -1
}
