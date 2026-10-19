package diagram

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// Type is a type of diagram, as Pele names it.
type Type string

// The types of diagram Mermaid knows.
const (
	Flowchart     Type = "flowchart"
	Sequence      Type = "sequence"
	Class         Type = "class"
	State         Type = "state"
	ER            Type = "er"
	Pie           Type = "pie"
	Gantt         Type = "gantt"
	GitGraph      Type = "gitGraph"
	Mindmap       Type = "mindmap"
	Timeline      Type = "timeline"
	Journey       Type = "journey"
	XYChart       Type = "xychart"
	QuadrantChart Type = "quadrantChart"
	Requirement   Type = "requirement"
	C4            Type = "c4"
	Sankey        Type = "sankey"
	Block         Type = "block"
	Packet        Type = "packet"
	Kanban        Type = "kanban"
	Architecture  Type = "architecture"
	Radar         Type = "radar"
	Treemap       Type = "treemap"
	Info          Type = "info"
	AgentFlow     Type = "agentflow"
	UseCase       Type = "usecase"
	Railroad      Type = "railroad"
	TreeView      Type = "treeView"
	Venn          Type = "venn"
	Cynefin       Type = "cynefin"
	Wardley       Type = "wardley"
	Swimlane      Type = "swimlane"
	Ishikawa      Type = "ishikawa"
	EventModeling Type = "eventmodeling"
)

// startsWith tests /^\s*word/, with fold for the i flag, which folds ASCII only.
func startsWith(s, word string, fold bool) (rest string, ok bool) {
	s = s[chars.SpaceRun(s, 0):]
	if len(s) < len(word) {
		return "", false
	}
	if fold && chars.EqualFoldASCII(s[:len(word)], word) || s[:len(word)] == word {
		return s[len(word):], true
	}
	return "", false
}

// wordEnds is \b after a word character: the next is none, or there is no next.
func wordEnds(rest string) bool { return rest == "" || !chars.IsWord(int(rest[0])) }

func prefix(word string) func(string) bool {
	return func(s string) bool {
		_, ok := startsWith(s, word, false)
		return ok
	}
}

func beta(word string, fold bool, after func(string) bool) func(string) bool {
	return func(s string) bool {
		rest, ok := startsWith(s, word, fold)
		return ok && after(rest)
	}
}

// Mermaid's detectors, in the order Mermaid registers them. The first match wins.
var detectors = []struct {
	typ   Type
	match func(string) bool
}{
	{Flowchart, prefix("flowchart-elk")},
	{Mindmap, prefix("mindmap")},
	{Architecture, prefix("architecture")},
	{AgentFlow, beta("agentflow-beta", false, wordEnds)},
	// /^\s*C4Context|C4Container|C4Component|C4Dynamic|C4Deployment/: only the first is held to
	// the start.
	{C4, func(s string) bool {
		if _, ok := startsWith(s, "C4Context", false); ok {
			return true
		}
		for _, w := range []string{"C4Container", "C4Component", "C4Dynamic", "C4Deployment"} {
			if strings.Contains(s, w) {
				return true
			}
		}
		return false
	}},
	{Kanban, prefix("kanban")},
	{Class, prefix("classDiagram")},
	{ER, prefix("erDiagram")},
	{Gantt, prefix("gantt")},
	{Info, prefix("info")},
	{Pie, prefix("pie")},
	{Requirement, prefix("requirement")},
	{Sequence, prefix("sequenceDiagram")},
	{Swimlane, beta("swimlane-beta", false, wordEnds)},
	{Flowchart, func(s string) bool { return prefix("graph")(s) || prefix("flowchart")(s) }},
	{Timeline, prefix("timeline")},
	{GitGraph, prefix("gitGraph")},
	{State, prefix("stateDiagram")},
	{Journey, prefix("journey")},
	{QuadrantChart, prefix("quadrantChart")},
	{Sankey, prefix("sankey")},
	{Packet, prefix("packet")},
	{XYChart, prefix("xychart")},
	{Block, prefix("block")},
	{EventModeling, prefix("eventmodeling")},
	{TreeView, prefix("treeView-beta")},
	{Radar, prefix("radar-beta")},
	// /^\s*ishikawa(-beta)?\b/i: the optional -beta gives way to the word's end.
	{Ishikawa, beta("ishikawa", true, wordEnds)},
	{Treemap, prefix("treemap")},
	// /^\s*railroad-(?:ebnf-|abnf-|peg-)?beta/i
	{Railroad, beta("railroad-", true, func(rest string) bool {
		for _, w := range []string{"beta", "ebnf-beta", "abnf-beta", "peg-beta"} {
			if chars.HasPrefixFold(rest, w) {
				return true
			}
		}
		return false
	})},
	{Venn, prefix("venn-beta")},
	{Wardley, beta("wardley-beta", true, func(string) bool { return true })},
	// /^\s*cynefin-beta(?:[\s:]|$)/
	{Cynefin, beta("cynefin-beta", false, func(rest string) bool {
		return rest == "" || rest[0] == ':' || chars.IsSpace(chars.Unit(rest, 0))
	})},
	// /^\s*usecase-beta(?:\s|$)/
	{UseCase, beta("usecase-beta", false, func(rest string) bool {
		return rest == "" || chars.IsSpace(chars.Unit(rest, 0))
	})},
}

// Detect is the type of diagram a preprocessed text is, as Pele's detect finds it, or "" when
// it is none.
func Detect(cleaned string) Type {
	for _, d := range detectors {
		if d.match(cleaned) {
			return d.typ
		}
	}
	return ""
}

// DetectType is the type of diagram a text is, or "" when it is none, as Pele's detectType
// finds it. Its error is one of its front matter. Bytes that are not UTF-8 are read as
// Prepare reads them.
func DetectType(text string) (Type, error) {
	pre, err := Preprocess(chars.Decode(text))
	if err != nil {
		return "", err
	}
	return Detect(pre.Text), nil
}
