package diagram

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them.

type peleError struct {
	Code    Code    `json:"code"`
	Message string  `json:"message"`
	Type    *string `json:"type"`
	Line    int     `json:"line"`
	Column  int     `json:"column"`
	Snippet string  `json:"snippet"`
}

type peleParse struct {
	Preprocessed []struct {
		Source       string     `json:"source"`
		Text         string     `json:"text"`
		WithComments *string    `json:"withComments"`
		Title        *string    `json:"title"`
		Config       any        `json:"config"`
		LineOffset   int        `json:"lineOffset"`
		Type         *string    `json:"type"`
		Error        *peleError `json:"error"`
	} `json:"preprocessed"`
	Encoded []struct {
		Text    string `json:"text"`
		Encoded string `json:"encoded"`
	} `json:"encoded"`
}

func load(t *testing.T, path string, v any) {
	t.Helper()
	if err := fixture.Load(path, v); err != nil {
		t.Fatal(err)
	}
}

// sameJSON reports whether v, written as JSON and read back, is want.
func sameJSON(v, want any) (bool, string) {
	data, err := json.Marshal(v)
	if err != nil {
		return false, err.Error()
	}
	var got any
	if err := json.Unmarshal(data, &got); err != nil {
		return false, err.Error()
	}
	return reflect.DeepEqual(got, want), string(data)
}

func sameError(err error, want *peleError) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	typ := ""
	if want.Type != nil {
		typ = *want.Type
	}
	return e.Code == want.Code && e.Message == want.Message && string(e.Type) == typ && e.Line == want.Line &&
		e.Column == want.Column && e.Snippet == want.Snippet
}

func TestPreprocessMatchesPele(t *testing.T) {
	var p peleParse
	load(t, "testdata/pele/parse.json", &p)
	if len(p.Preprocessed) < 1000 {
		t.Fatalf("only %d cases", len(p.Preprocessed))
	}
	failed := 0
	for _, c := range p.Preprocessed {
		got, err := preprocess(c.Source)
		if c.Error != nil {
			if !sameError(err, c.Error) {
				t.Errorf("preprocess(%q)\n got error %#v\nPele %#v", c.Source, err, c.Error)
				failed++
			}
			continue
		}
		if err != nil {
			t.Errorf("preprocess(%q): %v, Pele had none", c.Source, err)
			failed++
			continue
		}
		withComments := c.Text
		if c.WithComments != nil {
			withComments = *c.WithComments
		}
		title, hasTitle := "", c.Title != nil
		if hasTitle {
			title = *c.Title
		}
		typ := ""
		if c.Type != nil {
			typ = *c.Type
		}
		sameConfig, config := sameJSON(yaml.Encode(got.Config, true), c.Config)
		if got.Text != c.Text || got.WithComments != withComments || got.Title != title || got.HasTitle != hasTitle ||
			got.LineOffset != c.LineOffset || !sameConfig || string(detect(got.Text)) != typ {
			want, _ := json.Marshal(c)
			t.Errorf("preprocess(%q)\n got text %q, with comments %q, title %q (%v), line offset %d, type %q, config %s\nPele %s",
				c.Source, got.Text, got.WithComments, got.Title, got.HasTitle, got.LineOffset, detect(got.Text), config, want)
			failed++
		}
		if failed > 20 {
			t.Fatal("too many failures")
		}
	}
}

func TestEncodeEntitiesMatchesPele(t *testing.T) {
	var p peleParse
	load(t, "testdata/pele/parse.json", &p)
	if len(p.Encoded) < 1000 {
		t.Fatalf("only %d cases", len(p.Encoded))
	}
	for _, c := range p.Encoded {
		if got := encodeEntities(c.Text); got != c.Encoded {
			t.Errorf("encodeEntities(%q)\n got %q\nPele %q", c.Text, got, c.Encoded)
		}
	}
}

func TestPrepare(t *testing.T) {
	if _, err := Prepare("graph TD\nA", Options{Limit: 5}); err == nil || err.(*Error).Code != Limit ||
		err.Error() != "Diagram source is longer than the limit of 5 characters." {
		t.Errorf("limit: %v", err)
	}
	// Counted in UTF-16 code units: 😀 is two.
	if _, err := Prepare("pie 😀", Options{Limit: 5}); err == nil {
		t.Error("limit counted in bytes or characters")
	}
	if _, err := Prepare("pie 😀", Options{Limit: 6}); err != nil {
		t.Error(err)
	}
	if _, err := Prepare("nothing", Options{}); err == nil || err.(*Error).Code != UnsupportedDiagram {
		t.Errorf("unsupported: %v", err)
	}
	p, err := Prepare("---\ntitle: T\nconfig:\n  a: 1\n---\n%%{init: {\"b\": 2}}%%\ngraph TD\nA[\"#35;\"]", Options{Config: Config{"a": 0.0, "c": 3.0}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Type != Flowchart || p.Title != "T" || !p.HasTitle || p.LineOffset != 5 || p.Limits.MaxEdges != DefaultMaxEdges ||
		p.Config["a"] != 1.0 || p.Config["b"] != 2.0 || p.Config["c"] != 3.0 {
		t.Errorf("%+v", p)
	}
	if got := p.Source(false); got != "graph TD\nA[\"ﬂ°°35¶ß\"]\n" {
		t.Errorf("source %q", got)
	}
}
