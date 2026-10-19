package venn

import (
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// normalizeText takes one pair of quotes off an id, label or style value, as Mermaid does and
// as Pele's normalizeText does.
func normalizeText(text string) string {
	trimmed := chars.Trim(text)
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		return trimmed[1 : len(trimmed)-1]
	}
	return trimmed
}

// normalizeIds is the ids without their quotes, sorted as JavaScript's sort orders strings,
// as Pele's normalizeIds gives them.
func normalizeIds(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = normalizeText(id)
	}
	slices.SortStableFunc(out, compare16)
	return out
}

// compare16 compares two strings by their UTF-16 code units, as JavaScript's sort does, which
// orders characters above the Basic Multilingual Plane before those from U+E000 up, where
// UTF-8 orders them after.
func compare16(a, b string) int {
	ua, ub := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			if ua[i] < ub[i] {
				return -1
			}
			return 1
		}
	}
	return len(ua) - len(ub)
}

// db is Pele's VennDb: the model, with what the parser and the lexer keep while they read.
type db struct {
	Model
	known   map[string]bool
	current []string
	// indent is whether an indented `text` line belongs to the set or union before it. The
	// lexer reads it.
	indent bool
}

func (d *db) setDiagramTitle(title string) { d.Title = title }

// addSubsetData adds a set or an overlap. A set without a size gets 10, and an overlap of n
// sets 10 / n².
func (d *db) addSubsetData(ids []string, label *string, size *float64) {
	sets := normalizeIds(ids)
	d.current = sets
	if len(sets) == 1 {
		if d.known == nil {
			d.known = map[string]bool{}
		}
		d.known[sets[0]] = true
	}
	s := Subset{Sets: sets}
	if size != nil {
		s.Size = *size
	} else {
		// V8 works out ids.length ** 2 as a product.
		n := float64(len(ids))
		s.Size = 10 / float64(n*n)
	}
	s.Label = normalized(label)
	d.Subsets = append(d.Subsets, s)
}

// normalized is a label without its quotes, or nil for one that is missing or empty, as
// Pele's `label ? normalizeText(label) : undefined` gives it.
func normalized(label *string) *string {
	if label == nil || *label == "" {
		return nil
	}
	s := normalizeText(*label)
	return &s
}

func (d *db) addTextData(ids []string, id string, label *string) {
	d.TextNodes = append(d.TextNodes, Text{Sets: normalizeIds(ids), ID: normalizeText(id), Label: normalized(label)})
}

func (d *db) addStyleData(ids []string, data []Declaration) {
	var styles []Declaration
	for _, decl := range data {
		value := decl.Value
		if value != "" {
			value = normalizeText(value)
		}
		styles = setDeclaration(styles, decl.Property, value)
	}
	d.Styles = append(d.Styles, Style{Targets: normalizeIds(ids), Styles: styles})
}

// setDeclaration sets a property in a list of declarations as Map.set does: in its place if it
// is there, at the end if it is not.
func setDeclaration(list []Declaration, property, value string) []Declaration {
	for i := range list {
		if list[i].Property == property {
			list[i].Value = value
			return list
		}
	}
	return append(list, Declaration{property, value})
}

func (d *db) validateUnionIdentifiers(ids []string) error {
	var unknown []string
	for _, id := range ids {
		if id = normalizeText(id); !d.known[id] {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		return diagram.NewError("unknown set identifier: "+strings.Join(unknown, ", "), diagram.Semantic, "venn")
	}
	return nil
}

func (d *db) getCurrentSets() []string   { return d.current }
func (d *db) getIndentMode() bool        { return d.indent }
func (d *db) setIndentMode(enabled bool) { d.indent = enabled }
