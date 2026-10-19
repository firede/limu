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

// defaultShape is the shape of a node that names none, or one agent flows do not draw.
const defaultShape = "roundedRect"

// resolveShapeAlias is the shape an author-facing name stands for, or the name itself, as
// Pele's resolveShapeAlias gives it.
func resolveShapeAlias(shape string) string {
	if s, ok := shapeAliases[shape]; ok {
		return s
	}
	return shape
}

// isValidShape reports whether a node's metadata may name a shape so, as Pele's isValidShape
// does: a flowchart's shape, or "constructor", which Mermaid finds with `in` on a plain object.
func isValidShape(name string) bool {
	return flowchart.IsValidShape(name) || name == "constructor"
}

// normaliseNodeShapes reduces every node's shape to one agent flows draw, reporting the
// removed and unsupported ones, as Pele's normaliseNodeShapes does.
func normaliseNodeShapes(nodes []*graphNode, d *DB) {
	for _, node := range nodes {
		if node.IsGroup {
			continue
		}
		if node.Shape == "" || node.Shape == "squareRect" || node.Shape == "rect" {
			node.Shape = defaultShape
		}
		if removedShapes[node.Shape] {
			d.emitError(ShapeRemoved, `shape "`+node.Shape+`" was removed in v0.8.1, using "`+defaultShape+`"`, node.ID)
			node.Shape = defaultShape
		} else if !allowedShapes[node.Shape] {
			d.emitWarning(ShapeUnsupported, `shape "`+node.Shape+`" is not supported, using "`+defaultShape+`"`, node.ID)
			node.Shape = defaultShape
		}
	}
}
