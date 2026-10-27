package diagram

import "github.com/firede/limu/internal/chars"

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

// MermaidType is a type of diagram as Mermaid names it, as Pele's MermaidType is: the name
// Mermaid's detectType gives, such as "flowchart-v2" for `graph TD`. It is the Type, but for
// "flowchart-v2", "flowchart-elk", "classDiagram", "stateDiagram" and the railroad diagrams'
// "railroadEbnf", "railroadAbnf" and "railroadPeg".
type MermaidType string

// Detected is a diagram's type, as Pele's Detected is: as Pele names it and as Mermaid does.
type Detected struct {
	Type        Type
	MermaidType MermaidType
}

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

func always(string) bool { return true }

func beta(word string, fold bool, after func(string) bool) func(string) bool {
	return func(s string) bool {
		rest, ok := startsWith(s, word, fold)
		return ok && after(rest)
	}
}

// Mermaid's detectors, in the order Mermaid registers them. The first match wins.
var detectors = []struct {
	typ     Type
	mermaid MermaidType
	match   func(string) bool
}{
	{Flowchart, "flowchart-elk", prefix("flowchart-elk")},
	{Mindmap, "mindmap", prefix("mindmap")},
	{Architecture, "architecture", prefix("architecture")},
	{AgentFlow, "agentflow", beta("agentflow-beta", false, wordEnds)},
	// /^\s*(?:C4Context|C4Container|C4Component|C4Dynamic|C4Deployment)/
	{C4, "c4", func(s string) bool {
		for _, w := range []string{"C4Context", "C4Container", "C4Component", "C4Dynamic", "C4Deployment"} {
			if _, ok := startsWith(s, w, false); ok {
				return true
			}
		}
		return false
	}},
	{Kanban, "kanban", prefix("kanban")},
	{Class, "classDiagram", prefix("classDiagram")},
	{ER, "er", prefix("erDiagram")},
	{Gantt, "gantt", prefix("gantt")},
	{Info, "info", prefix("info")},
	{Pie, "pie", prefix("pie")},
	{Requirement, "requirement", prefix("requirement")},
	{Sequence, "sequence", prefix("sequenceDiagram")},
	{Swimlane, "swimlane", beta("swimlane-beta", false, wordEnds)},
	{Flowchart, "flowchart-v2", func(s string) bool { return prefix("graph")(s) || prefix("flowchart")(s) }},
	{Timeline, "timeline", prefix("timeline")},
	{GitGraph, "gitGraph", prefix("gitGraph")},
	{State, "stateDiagram", prefix("stateDiagram")},
	{Journey, "journey", prefix("journey")},
	{QuadrantChart, "quadrantChart", prefix("quadrantChart")},
	{Sankey, "sankey", prefix("sankey")},
	{Packet, "packet", prefix("packet")},
	{XYChart, "xychart", prefix("xychart")},
	{Block, "block", prefix("block")},
	{EventModeling, "eventmodeling", prefix("eventmodeling")},
	{TreeView, "treeView", prefix("treeView-beta")},
	{Radar, "radar", prefix("radar-beta")},
	// /^\s*ishikawa(-beta)?\b/i: the optional -beta gives way to the word's end.
	{Ishikawa, "ishikawa", beta("ishikawa", true, wordEnds)},
	{Treemap, "treemap", prefix("treemap")},
	// /^\s*railroad-beta/i, and one for each dialect, as /^\s*railroad-ebnf-beta/i, with a name of
	// its own.
	{Railroad, "railroad", beta("railroad-beta", true, always)},
	{Railroad, "railroadEbnf", beta("railroad-ebnf-beta", true, always)},
	{Railroad, "railroadAbnf", beta("railroad-abnf-beta", true, always)},
	{Railroad, "railroadPeg", beta("railroad-peg-beta", true, always)},
	{Venn, "venn", prefix("venn-beta")},
	{Wardley, "wardley", beta("wardley-beta", true, always)},
	// /^\s*cynefin-beta(?:[\s:]|$)/
	{Cynefin, "cynefin", beta("cynefin-beta", false, func(rest string) bool {
		return rest == "" || rest[0] == ':' || chars.IsSpace(chars.Unit(rest, 0))
	})},
	// /^\s*usecase-beta(?:\s|$)/
	{UseCase, "usecase", beta("usecase-beta", false, func(rest string) bool {
		return rest == "" || chars.IsSpace(chars.Unit(rest, 0))
	})},
}

// detectDiagram is the type of diagram a preprocessed text is, as Pele's detectDiagram finds
// it, and false when it is none.
func detectDiagram(cleaned string) (Detected, bool) {
	for _, d := range detectors {
		if d.match(cleaned) {
			return Detected{d.typ, d.mermaid}, true
		}
	}
	return Detected{}, false
}

// detect is the type of diagram a preprocessed text is, as Pele's detect finds it, or "" when
// it is none.
func detect(cleaned string) Type {
	found, _ := detectDiagram(cleaned)
	return found.Type
}

// DetectType is the type of diagram a text is, or "" when it is none, as Pele's detectType
// finds it. Its error is one of its front matter. Bytes that are not UTF-8 are read as
// Prepare reads them.
func DetectType(text string) (Type, error) {
	pre, err := preprocess(chars.Decode(text), nil)
	if err != nil {
		return "", err
	}
	return detect(pre.Text), nil
}
