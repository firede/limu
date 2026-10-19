package pelesvg

import (
	"fmt"
	"strings"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// CapJoin is the cap and the join an item is stroked with, its author's style laid over its
// own, as the render oracles write them: "round miter".
func CapJoin(it *scene.Item) string {
	var look *style.Look
	r := look.Resolve(string(it.Role), it.Style, it.Author)
	return r.Cap.String() + " " + r.Join.String()
}

// FlowLines writes the scene of a flowchart or a swimlane diagram as the render oracle reads
// Pele's SVG of one: a line for each thing drawn, in the order drawn, with numbers as Pele's
// SVG writes them.
func FlowLines(sc *scene.Scene) []string {
	num := Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect, cx, cy float64) string {
		return pt(r.X-cx, r.Y-cy) + "," + num(r.W) + "," + num(r.H)
	}
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on (cx, cy).
	textAt := func(l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		return "t " + pt(cx, cy-l.Height/2+l.LineHeight/2+shift)
	}
	label := func(l *text.Label, cx, cy float64) {
		if l != nil && len(l.Lines) > 0 {
			lines = append(lines, textAt(l, cx, cy))
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		id := ID(it.ID)
		classes := strings.Join(it.Class, " ")
		switch it.Role {
		case "flowchart.title":
			if it.Label != nil && len(it.Label.Lines) > 0 {
				lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
			}
		case "flowchart.cluster":
			lines = append(lines, fmt.Sprintf("cluster %s [%s] %s", id, classes, box(it.Box, 0, 0)))
			label(it.Label, it.At.X, it.At.Y)
		case "swimlane.lane":
			lines = append(lines, fmt.Sprintf("lane %s [%s] %s band %s", id, classes, box(it.Parts[0].Box, 0, 0), box(it.Parts[1].Box, 0, 0)))
			label(it.Label, it.At.X, it.At.Y)
			if it.Label != nil && len(it.Label.Lines) > 0 && it.Rotate != 0 {
				lines = append(lines, "rotate "+num(it.Rotate)+" "+num(it.At.X)+" "+num(it.At.Y))
			}
		case "flowchart.edge":
			flags := ""
			if w, ok := it.Style.StrokeWidth.Get(); ok && w == 2.5 {
				flags += " thick"
			}
			if it.Style.Dash.Set() {
				flags += " dotted"
			}
			lines = append(lines, fmt.Sprintf("edge %s [%s]%s %s", id, classes, flags, D(it.Parts[0].Path)))
			lines = append(lines, "stroke "+CapJoin(it))
			for _, m := range it.Ends {
				if s := Marker(m); s != "" {
					lines = append(lines, "m "+s)
				}
			}
		case "flowchart.cluster-title":
			lines = append(lines, "struck "+id)
			label(it.Label, it.At.X, it.At.Y)
		case "flowchart.edge-label":
			lines = append(lines, "label "+id+" "+box(it.Box, 0, 0))
			label(it.Label, it.At.X, it.At.Y)
		case "flowchart.node":
			cx, cy := it.Box.X+it.Box.W/2, it.Box.Y+it.Box.H/2
			lines = append(lines, fmt.Sprintf("node %s %s [%s] translate(%s)", id, it.Shape, classes, pt(cx, cy)))
			if pic := it.Picture; pic != nil && pic.Image != "" {
				lines = append(lines, "image "+pic.Image+" "+box(pic.Box, cx, cy))
			}
			if pic := it.Picture; pic != nil && pic.Icon != "" {
				lines = append(lines, "icon "+ID(pic.Icon)+" "+box(pic.Box, cx, cy))
			}
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			label(it.Label, it.At.X-cx, it.At.Y-cy)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// ID is an id as Pele writes a data-id: entities decoded, characters XML does not allow
// dropped.
func ID(id string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, text.DecodeEntities(id))
}

// Marker is the outline Pele's marker draws, as the render oracle reads it.
func Marker(m scene.Marker) string {
	const arrow = scene.MarkerSize
	num := Num
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	switch m.Type {
	case "arrow_point":
		bx, by := x-float64(dx*arrow), y-float64(dy*arrow)
		half := float64(arrow * 0.42)
		return "M" + num(x) + "," + num(y) + "L" + num(bx+float64(nx*half)) + "," + num(by+float64(ny*half)) +
			"L" + num(bx-float64(nx*half)) + "," + num(by-float64(ny*half)) + "Z"
	case "arrow_circle":
		return "o " + num(x-float64(dx*4)) + "," + num(y-float64(dy*4))
	case "arrow_cross":
		cx, cy := x-float64(dx*4), y-float64(dy*4)
		const a = 3.5
		ux, uy := float64((dx+nx)*a), float64((dy+ny)*a)
		vx, vy := float64((dx-nx)*a), float64((dy-ny)*a)
		return "M" + num(cx-ux) + "," + num(cy-uy) + "L" + num(cx+ux) + "," + num(cy+uy) +
			"M" + num(cx-vx) + "," + num(cy-vy) + "L" + num(cx+vx) + "," + num(cy+vy)
	}
	return ""
}

// FirstDiff is the first line where two lists of lines differ, or -1 where they are the same.
func FirstDiff(a, b []string) int {
	for i := range max(len(a), len(b)) {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			return i
		}
	}
	return -1
}

// At is the line i of lines, or "(none)" past their end.
func At(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(none)"
}
