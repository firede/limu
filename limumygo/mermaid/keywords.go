package mermaid

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"

	"github.com/firede/limu/diagram"
)

// keywordsJSON holds the keywords of each type, which the website's highlighter reads too
// (src/lib/highlight.ts of firede/limu-website, from the release the site follows), so that
// both highlight the same words:
//
//   - starts: the words that start a statement of each type, as Pele's playground lists
//     them for completion (src/lib/playground-completions.ts, keywords);
//   - inline: words that are keywords wherever they are in a statement of the type, not
//     only at its start;
//   - common: the words that start a statement of every type;
//   - directions: the ways a diagram runs, highlighted as constants.
//
// Each list is words separated by spaces.
//
//go:embed keywords.json
var keywordsJSON []byte

type keywords struct {
	starts, inline map[string]bool
}

type keywordFile struct {
	Common     string            `json:"common"`
	Directions string            `json:"directions"`
	Starts     map[string]string `json:"starts"`
	Inline     map[string]string `json:"inline"`
}

func set(words string) map[string]bool {
	m := map[string]bool{}
	for w := range strings.FieldsSeq(words) {
		m[w] = true
	}
	return m
}

var file = sync.OnceValue(func() keywordFile {
	var f keywordFile
	if err := json.Unmarshal(keywordsJSON, &f); err != nil {
		panic("mermaid: keywords.json: " + err.Error())
	}
	return f
})

var tables = sync.OnceValue(func() map[diagram.Type]keywords {
	f := file()
	t := map[diagram.Type]keywords{}
	var all strings.Builder
	for typ, words := range f.Starts {
		t[diagram.Type(typ)] = keywords{set(words + " " + f.Common), set(f.Inline[typ])}
		all.WriteString(words + " ")
	}
	// Text whose type is not known yet, or not read, gets every type's words.
	t[""] = keywords{set(all.String() + f.Common), nil}
	return t
})

// keywordsOf is the keywords of a type, or of every type for "".
func keywordsOf(t diagram.Type) keywords {
	if k, ok := tables()[t]; ok {
		return k
	}
	return tables()[""]
}

// directions are the ways a diagram runs, highlighted as constants.
var directions = sync.OnceValue(func() map[string]bool { return set(file().Directions) })
