package fixture

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The digests are those internal/oracle/run.mjs writes, as Node computes them.
func TestDigest(t *testing.T) {
	for _, c := range []struct {
		lines []string
		want  string
	}{
		{nil, "e3b0c44298fc1c149afbf4c8996fb924"},
		{[]string{"size 1,2", "t 3,4 é 😀", ""}, "834aae1f3304de4651cb95866c13e7a6"},
		// A lone surrogate, which JSON escapes and both Node and Go's encoding/json read as
		// U+FFFD.
		{[]string{"�"}, "8d75cfafa290dea108e554948eae67ba"},
	} {
		if got := Digest(c.lines); got != c.want {
			t.Errorf("Digest(%q) = %s, Node %s", c.lines, got, c.want)
		}
	}
}

// writeFixture writes v as the fixture at path, gzipped, as the oracle writes one.
func writeFixture(t *testing.T, path string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()
	if err := os.WriteFile(path+".gz", b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiff(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "flowchart")
	if err := os.MkdirAll(filepath.Join(dir, "testdata", "pele"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	const path = "testdata/pele/render.json"
	pele := map[string]string{"version": Pinned.Version, "commit": Pinned.Commit}
	// read moves a tag last, as the tests of flowcharts read Pele's lines, and the suite keeps
	// the line it moves one in.
	lines := []string{"size 10,10", "node a [clickable x]", "edge a-b M0,0"}
	read := func(line string) string { return strings.Replace(line, "[clickable x]", "[x clickable]", 1) }
	got := []string{"size 10,10", "node a [x clickable]", "edge a-b M0,0"}
	other := []string{"size 10,10", "node a [x clickable]", "edge a-b M1,1"}
	cases := []map[string]any{
		{"src": "a", "lines": lines},
		{"src": "b", "digest": Digest(lines), "kept": map[string]string{"1": lines[1]}},
	}
	writeFixture(t, path, map[string]any{"pele": pele, "cases": cases})
	var v struct {
		Cases []struct {
			Src string `json:"src"`
			Drawing
		} `json:"cases"`
	}
	f, err := LoadDrawings(path, &v)
	if err != nil {
		t.Fatal(err)
	}
	full, digest := &v.Cases[0].Drawing, &v.Cases[1].Drawing
	if got := digest.Line(1); got != lines[1] {
		t.Errorf("Line(1) = %q, not the line kept", got)
	}
	if got := digest.Line(2); got != "" {
		t.Errorf("Line(2) = %q, of a line not kept", got)
	}
	for _, c := range []struct {
		name string
		d    *Drawing
		got  []string
		want string
	}{
		{"lines, the same", full, got, ""},
		{"lines, differing", full, other, "line 2\n got edge a-b M1,1\nPele edge a-b M0,0"},
		{"digest, the same", digest, got, ""},
		{"digest, one line more", digest, append(got, "x"), "only a digest"},
		{"digest, a kept line differing", digest, []string{"size 10,10", "node a", "edge a-b M0,0"},
			"./scripts/oracle.sh --full <pele> flowchart-render writes every line to testdata/pele/render.full.json.gz, to show the first that differs. Of the lines it holds, this one differs: line 1\n got node a\nPele node a [x clickable]"},
		{"digest, differing", digest, other, "./scripts/oracle.sh --full <pele> flowchart-render writes every line"},
	} {
		if msg := f.Diff(1, c.d, c.got, read); c.want == "" && msg != "" || !strings.Contains(msg, c.want) {
			t.Errorf("%s: Diff = %q, not %q", c.name, msg, c.want)
		}
	}

	// The full file holds the lines of the digest, and shows which differs.
	writeFixture(t, "testdata/pele/render.full.json", map[string]any{"pele": pele, "cases": []map[string]any{
		{"src": "a", "lines": lines}, {"src": "b", "lines": lines},
	}})
	f, _ = LoadDrawings(path, &v)
	if msg, want := f.Diff(1, digest, other, read), "line 2\n got edge a-b M1,1\nPele edge a-b M0,0\n(Pele's lines from testdata/pele/render.full.json.gz)"; msg != want {
		t.Errorf("Diff with the full file = %q, not %q", msg, want)
	}
	// read changes a line the suite does not keep: the lines of the full file match.
	changes := func(line string) string { return read(strings.ReplaceAll(line, "M0,0", "M1,1")) }
	if msg, want := f.Diff(1, digest, other, changes), "read changes a line"; !strings.Contains(msg, want) {
		t.Errorf("Diff with a line read changes = %q, not %q", msg, want)
	}

	// A full file that holds other lines is stale.
	writeFixture(t, "testdata/pele/render.full.json", map[string]any{"pele": pele, "cases": []map[string]any{
		{"src": "a", "lines": lines}, {"src": "b", "lines": lines[:2]},
	}})
	f, _ = LoadDrawings(path, &v)
	if msg, want := f.Diff(1, digest, other, read), "it is stale, and ./scripts/oracle.sh --full <pele> flowchart-render writes it again"; !strings.Contains(msg, want) {
		t.Errorf("Diff with a stale full file = %q, not %q", msg, want)
	}
}
