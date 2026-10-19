package flowchart

// shapes are Mermaid's documented shape names: the short name first, then its aliases.
var shapes = [][]string{
	{"rect", "proc", "process", "rectangle", "squareRect"},
	{"rounded", "event", "roundedRect"},
	{"stadium", "terminal", "pill"},
	{"fr-rect", "subprocess", "subproc", "framed-rectangle", "subroutine"},
	{"cyl", "db", "database", "cylinder"},
	{"datastore", "data-store"},
	{"folder", "directory"},
	{"bucket"},
	{"console"},
	{"browser"},
	{"person"},
	{"circle", "circ"},
	{"bang"},
	{"cloud"},
	{"diam", "decision", "diamond", "question"},
	{"hex", "hexagon", "prepare"},
	{"lean-r", "lean-right", "in-out", "lean_right"},
	{"lean-l", "lean-left", "out-in", "lean_left"},
	{"trap-b", "priority", "trapezoid-bottom", "trapezoid"},
	{"trap-t", "manual", "trapezoid-top", "inv-trapezoid", "inv_trapezoid"},
	{"dbl-circ", "double-circle", "doublecircle"},
	{"text"},
	{"notch-rect", "card", "notched-rectangle"},
	{"lin-rect", "lined-rectangle", "lined-process", "lin-proc", "shaded-process"},
	{"sm-circ", "start", "small-circle", "stateStart"},
	{"fr-circ", "stop", "framed-circle", "stateEnd"},
	{"fork", "join", "forkJoin"},
	{"hourglass", "collate"},
	{"brace", "comment", "brace-l"},
	{"brace-r"},
	{"braces"},
	{"bolt", "com-link", "lightning-bolt"},
	{"doc", "document"},
	{"delay", "half-rounded-rectangle"},
	{"h-cyl", "das", "horizontal-cylinder"},
	{"lin-cyl", "disk", "lined-cylinder"},
	{"curv-trap", "curved-trapezoid", "display"},
	{"div-rect", "div-proc", "divided-rectangle", "divided-process"},
	{"tri", "extract", "triangle"},
	{"win-pane", "internal-storage", "window-pane"},
	{"f-circ", "junction", "filled-circle"},
	{"notch-pent", "loop-limit", "notched-pentagon"},
	{"flip-tri", "manual-file", "flipped-triangle"},
	{"sl-rect", "manual-input", "sloped-rectangle"},
	{"docs", "documents", "st-doc", "stacked-document"},
	{"st-rect", "procs", "processes", "stacked-rectangle"},
	{"bow-rect", "stored-data", "bow-tie-rectangle"},
	{"cross-circ", "summary", "crossed-circle"},
	{"tag-doc", "tagged-document"},
	{"tag-rect", "tagged-rectangle", "tag-proc", "tagged-process"},
	{"flag", "paper-tape"},
	{"odd", "rect_left_inv_arrow"},
	{"lin-doc", "lined-document"},
}

// internalShapes are the shapes Mermaid accepts by name without documenting them.
var internalShapes = []string{
	"state",
	"choice",
	"note",
	"composite",
	"rectWithTitle",
	"labelRect",
	"block_arrow",
	"collapsedGroup",
	"iconSquare",
	"iconCircle",
	"icon",
	"iconRounded",
	"imageSquare",
	"anchor",
	"kanbanItem",
	"mindmapCircle",
	"defaultMindmapNode",
	"classBox",
	"erBox",
	"requirementBox",
	"usecaseActor",
	"usecaseActorHollow",
	"usecaseActorAwesome",
	"usecaseActorIcon",
	"usecaseBusiness",
	"usecaseEllipse",
	"usecaseJsonTable",
}

// syntaxShapes are the shapes the bracket syntax gives, by their documented names.
var syntaxShapes = map[string]string{
	"square":  "rect",
	"round":   "rounded",
	"ellipse": "ellipse",
	"hexagon": "hex",
	"diamond": "diam",
}

var canonical = func() map[string]string {
	m := map[string]string{}
	for _, names := range shapes {
		for _, name := range names {
			m[name] = names[0]
		}
	}
	for _, name := range internalShapes {
		m[name] = name
	}
	return m
}()

// IsValidShape reports whether a node's metadata may name a shape so, as Pele's isValidShape
// does.
func IsValidShape(name string) bool {
	_, ok := canonical[name]
	return ok
}

// CanonicalShape is the documented name of a shape, as Pele's canonicalShape gives it: rect
// for one of no such name, or none.
func CanonicalShape(name string) string {
	if s, ok := syntaxShapes[name]; ok {
		return s
	}
	if s, ok := canonical[name]; ok {
		return s
	}
	return "rect"
}
