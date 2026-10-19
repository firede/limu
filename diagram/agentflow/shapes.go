package agentflow

import "github.com/firede/limu/diagram/flowchart"

// shapeAliases are the author-facing names of the shapes that carry meaning in an agent flow.
var shapeAliases = map[string]string{
	"task":     "roundedRect",
	"tool":     "subroutine",
	"input":    "lean-right",
	"decision": "diamond",
	"refdoc":   "lin-doc",
	"action":   "hexagon",
	"round":    "rect",
}

// removedShapes are the flowchart shapes agent flows no longer draw.
var removedShapes = map[string]bool{
	"doc": true, "stadium": true, "terminal": true, "circle": true, "trapezoid": true, "inv_trapezoid": true,
	"inv-trapezoid": true, "doublecircle": true, "double-circle": true, "typeDeclaration": true, "procs": true,
	"lean_left": true, "lean-left": true, "in-out": true, "cylinder": true, "ellipse": true, "odd": true,
	"tag-rect": true, "tagged-rectangle": true, "delay": true, "half-rounded-rectangle": true, "lin-rect": true,
	"lined-rectangle": true, "win-pane": true, "window-pane": true, "curv-trap": true, "curved-trapezoid": true,
}

// allowedShapes are the shapes agent flows draw.
var allowedShapes = map[string]bool{
	"roundedRect": true, "subroutine": true, "subprocess": true, "subproc": true, "framed-rectangle": true,
	"lean-right": true, "diamond": true, "lin-doc": true, "lined-document": true, "hexagon": true, "hex": true,
	"connector": true, "collapsedGroup": true,
}

// DefaultShape is the shape of a node that names none, or one agent flows do not draw.
const DefaultShape = "roundedRect"

// ResolveShapeAlias is the shape an author-facing name stands for, or the name itself, as
// Pele's resolveShapeAlias gives it.
func ResolveShapeAlias(shape string) string {
	if s, ok := shapeAliases[shape]; ok {
		return s
	}
	return shape
}

// IsValidShape reports whether a node's metadata may name a shape so, as Pele's isValidShape
// does: a flowchart's shape, or "constructor", which Mermaid finds with `in` on a plain object.
func IsValidShape(name string) bool {
	return flowchart.IsValidShape(name) || name == "constructor"
}

// normaliseNodeShapes reduces every node's shape to one agent flows draw, reporting the
// removed and unsupported ones, as Pele's normaliseNodeShapes does.
func normaliseNodeShapes(nodes []*GraphNode, d *DB) {
	for _, node := range nodes {
		if node.IsGroup {
			continue
		}
		if node.Shape == "" || node.Shape == "squareRect" || node.Shape == "rect" {
			node.Shape = DefaultShape
		}
		if removedShapes[node.Shape] {
			d.EmitError(ShapeRemoved, `shape "`+node.Shape+`" was removed in v0.8.1, using "`+DefaultShape+`"`, node.ID)
			node.Shape = DefaultShape
		} else if !allowedShapes[node.Shape] {
			d.EmitWarning(ShapeUnsupported, `shape "`+node.Shape+`" is not supported, using "`+DefaultShape+`"`, node.ID)
			node.Shape = DefaultShape
		}
	}
}
