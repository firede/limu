package draw

import (
	"math"
	"slices"

	"github.com/firede/limu/diagram/internal/graphs"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/text"
)

// Arrow is the length of an arrowhead.
const Arrow = scene.MarkerSize

const (
	// Corner is the most a corner is rounded by in a stepped route.
	Corner = 12
	// Handle is how far the handles of a wide turn reach along its two legs. A little more
	// than a circle's, which makes the turn fuller.
	Handle = 0.62
	// Lead is how far an edge runs straight before a marker at its end, where it has the
	// room.
	Lead = 6
)

// handleRest is what is left of a leg past a handle, 1 − Handle, worked out as JavaScript
// works it out.
var handleRest = 1 - float64(Handle)

// An EdgePath is the path an edge is drawn along, and where its markers go: the tip of the
// one at its start and the way it points, and the same for its end.
type EdgePath struct {
	Path             *scene.Path
	SX, SY, SDX, SDY float64
	EX, EY, EDX, EDY float64
}

// Ends are markers of the types at the ends of the path. The names Pele's models give an end
// with no marker, "none" and "arrow_open" (the end of an open line), are the empty type, as
// the scene has it.
func (e EdgePath) Ends(start, end scene.MarkerType) [2]scene.Marker {
	return [2]scene.Marker{
		{Type: drawn(start), At: scene.Point{X: e.SX, Y: e.SY}, DX: e.SDX, DY: e.SDY},
		{Type: drawn(end), At: scene.Point{X: e.EX, Y: e.EY}, DX: e.EDX, DY: e.EDY},
	}
}

// drawn is the type of a marker that is drawn, or the empty type for one of Pele's names for
// none.
func drawn(t scene.MarkerType) scene.MarkerType {
	if t == "none" || t == "arrow_open" {
		return ""
	}
	return t
}

// A SideReach says how far from each side of each node the edges that meet it start to
// turn: as far as the largest marker there needs, for every edge on the side, so that none
// turns sooner than its neighbour and crosses it. It is Pele's sideReach. Its zero value is
// ready to use.
type SideReach struct {
	reach map[[2]int]float64
}

// Add takes the marker of an edge's end on a side of a node (0 top, 1 right, 2 bottom,
// 3 left; -1, none, is left out).
func (r *SideReach) Add(node, side int, t scene.MarkerType) {
	trim := t.Trim()
	key := [2]int{node, side}
	if side >= 0 && trim > 0 && trim+Lead > r.reach[key] {
		if r.reach == nil {
			r.reach = map[[2]int]float64{}
		}
		r.reach[key] = trim + Lead
	}
}

// Of is how far from the side of the node its edges start to turn.
func (r *SideReach) Of(node, side int) float64 {
	if side < 0 {
		return 0
	}
	return r.reach[[2]int{node, side}]
}

// OwnReach is how far from its node an edge starts to turn when nothing beside it needs it
// to turn further out: far enough to clear its own marker, trimmed by trim. It is what
// Pele's routePath takes when it is given no reach.
func OwnReach(trim float64) float64 {
	if trim > 0 {
		return trim + Lead
	}
	return 0
}

// RoutePath is the path of a route of x, y, axis triples, trimmed at each end to leave
// room for markers, as Pele's routePath draws it. Between two points that are not in line,
// a route is one S that leaves and arrives along the flow, or a step in a stepped route.
// Where the route itself turns a corner, a stepped route rounds it a little, and any other
// but a linear one turns as widely as it can: the turn takes the whole of a leg that ends at
// a node and half of a leg it shares with the next turn. The curve is Mermaid's: "linear",
// "step", "stepBefore" and "stepAfter" draw as they say, and any other, or none, is smooth.
// startReach and endReach say how far from its nodes the edge starts to turn, where the
// other edges on the same side of a node need it to; OwnReach gives what Pele takes when
// none is given. The route has at least two points.
func RoutePath(route []float64, curve string, startTrim, endTrim, startReach, endReach float64) EdgePath {
	linear := curve == "linear"
	stepped := curve == "step" || curve == "stepBefore" || curve == "stepAfter"
	var xs, ys, axes []float64
	near := func(ax, ay, bx, by float64) bool { return math.Abs(ax-bx) < 0.01 && math.Abs(ay-by) < 0.01 }
	for i := 0; i+2 < len(route); i += 3 {
		x, y, axis := route[i], route[i+1], route[i+2]
		last := len(xs) - 1
		// A stepped route turns at right angles where a smooth one would curve.
		if stepped && last >= 0 && math.Abs(x-xs[last]) >= 0.01 && math.Abs(y-ys[last]) >= 0.01 {
			px, py := xs[last], ys[last]
			var mx, my float64
			switch curve {
			case "stepBefore":
				mx, my = px, py
			case "stepAfter":
				mx, my = x, y
			default:
				mx, my = (px+x)/2, (py+y)/2
			}
			turns := [4]float64{mx, py, mx, y}
			if axis == 0 {
				turns = [4]float64{px, my, x, my}
			}
			for k := 0; k < 4; k += 2 {
				if near(turns[k], turns[k+1], px, py) || near(turns[k], turns[k+1], x, y) {
					continue
				}
				xs = append(xs, turns[k])
				ys = append(ys, turns[k+1])
				axes = append(axes, axis)
			}
		}
		xs = append(xs, x)
		ys = append(ys, y)
		axes = append(axes, axis)
	}
	last := len(xs) - 1
	aligned := func(i int) bool { return math.Abs(xs[i]-xs[i-1]) < 0.01 || math.Abs(ys[i]-ys[i-1]) < 0.01 }
	straight := func(i int) bool { return linear || aligned(i) }
	// The way the route heads from point i to point j: along the flow where an S joins them.
	unit := func(i, j int, axis float64) (float64, float64) {
		if !straight(max(i, j)) {
			if axis == 0 {
				return 0, jsmath.SignOr1(ys[j] - ys[i])
			}
			return jsmath.SignOr1(xs[j] - xs[i]), 0
		}
		dx, dy := xs[j]-xs[i], ys[j]-ys[i]
		l := jsmath.Or1(jsmath.Hypot(dx, dy))
		return dx / l, dy / l
	}
	sdx, sdy := unit(0, 1, axes[1])
	edx, edy := unit(last-1, last, axes[last])
	sx, sy, ex, ey := xs[0], ys[0], xs[last], ys[last]
	xs[0] += float64(sdx * startTrim)
	ys[0] += float64(sdy * startTrim)
	xs[last] -= float64(edx * endTrim)
	ys[last] -= float64(edy * endTrim)
	length := func(i int) float64 { return math.Abs(xs[i]-xs[i-1]) + math.Abs(ys[i]-ys[i-1]) }
	// Whether the route turns a corner at point i, from running one way to running the other.
	turns := func(i int) bool {
		return !linear && i > 0 && i < last && aligned(i) && aligned(i+1) && length(i) >= 0.01 && length(i+1) >= 0.01 &&
			(math.Abs(xs[i]-xs[i-1]) < 0.01) != (math.Abs(xs[i+1]-xs[i]) < 0.01)
	}
	// An edge that curves into a marker runs straight for a little way first, where it has
	// the room, so that the marker does not sit on the bend.
	along := func(i int, ux, uy float64) float64 {
		return float64(math.Abs(xs[i]-xs[i-1])*math.Abs(ux)) + float64(math.Abs(ys[i]-ys[i-1])*math.Abs(uy))
	}
	// Of the way from the end of a run to where it turns, the part that makes up for a
	// larger marker beside this one is kept where it can be, and the rest is at most a
	// quarter of what is left, so that ends on one side of a node turn equally far from it.
	lead := func(reach, trim, run float64) float64 {
		room := min(max(0, reach-Lead-trim), run/4)
		return room + min(reach-trim-room, (run-room)/4)
	}
	var arrives, leaves float64
	if endReach > endTrim && (!straight(last) || turns(last-1)) {
		arrives = lead(endReach, endTrim, along(last, edx, edy))
	}
	if startReach > startTrim && (!straight(1) || turns(1)) {
		leaves = lead(startReach, startTrim, along(1, sdx, sdy))
	}
	if arrives >= 1 {
		xs = slices.Insert(xs, last, xs[last]-float64(edx*arrives))
		ys = slices.Insert(ys, last, ys[last]-float64(edy*arrives))
		axes = slices.Insert(axes, last, axes[last])
		last++
	}
	if leaves >= 1 {
		xs = slices.Insert(xs, 1, xs[0]+float64(sdx*leaves))
		ys = slices.Insert(ys, 1, ys[0]+float64(sdy*leaves))
		axes = slices.Insert(axes, 1, axes[1])
		last++
	}

	p := new(scene.Path).MoveTo(xs[0], ys[0])
	atX, atY := xs[0], ys[0]
	lineTo := func(x, y float64) {
		if near(x, y, atX, atY) {
			return
		}
		p.LineTo(x, y)
		atX, atY = x, y
	}
	for i := 1; i <= last; i++ {
		x, y := xs[i], ys[i]
		if !straight(i) {
			down := axes[i] == 0
			run, across := x-atX, y-atY
			if down {
				run, across = across, run
			}
			k := float64(math.Abs(run) * graphs.Ease(across, run))
			var ux, uy float64
			if down {
				uy = float64(jsmath.Sign(run) * k)
			} else {
				ux = float64(jsmath.Sign(run) * k)
			}
			p.CubeTo(atX+ux, atY+uy, x-ux, y-uy, x, y)
			atX, atY = x, y
			continue
		}
		if !turns(i) {
			// A point part of the way along one straight run adds nothing to the path.
			if i < last && aligned(i) && aligned(i+1) && float64((x-xs[i-1])*(xs[i+1]-x))+float64((y-ys[i-1])*(ys[i+1]-y)) > 0 {
				continue
			}
			lineTo(x, y)
			continue
		}
		// How much of the leg before the turn and of the leg after it the turn takes: all of
		// one that ends at a node, short of the straight run into a marker, and half of one
		// it shares.
		before, after := length(i), length(i+1)
		if turns(i - 1) {
			before = length(i) / 2
		}
		if turns(i + 1) {
			after = length(i+1) / 2
		}
		if stepped {
			before = min(Corner, before, after)
			after = before
		}
		ux, uy := unit(i-1, i, axes[i])
		vx, vy := unit(i, i+1, axes[i+1])
		lineTo(x-float64(ux*before), y-float64(uy*before))
		atX, atY = x+float64(vx*after), y+float64(vy*after)
		if stepped {
			p.QuadTo(x, y, atX, atY)
		} else {
			p.CubeTo(x-float64(ux*before*handleRest), y-float64(uy*before*handleRest),
				x+float64(vx*after*handleRest), y+float64(vy*after*handleRest), atX, atY)
		}
	}
	return EdgePath{p, sx, sy, -sdx, -sdy, ex, ey, edx, edy}
}

// LoopPath is a self-loop beside a node centered on cx, cy and drawn w by h, as Pele's
// loopPath draws it: out of the side that lies across the flow and back in. k counts the
// loops already on the node, so each one reaches a little further; base is how far the
// first reaches, and trim is the marker's at its end. It also gives where the edge's label,
// labelW by labelH, is centered: beyond the loop.
func LoopPath(cx, cy, w, h float64, k int, sideways bool, base, trim, labelW, labelH float64) (EdgePath, scene.Point) {
	kf := float64(k)
	half := h / 2
	if sideways {
		half = w / 2
	}
	spread := min(half-4, 8+float64(kf*6))
	reach := base + float64(kf*8)
	l := jsmath.Or1(jsmath.Hypot(reach, spread))
	// Local frame: out points away from the node, along runs along its side.
	at := func(out, along float64) (float64, float64) {
		if sideways {
			return cx + along, cy + h/2 + out
		}
		return cx + w/2 + out, cy + along
	}
	tx := float64((reach / l) * trim)
	ty := float64((spread / l) * trim)
	var label scene.Point
	if sideways {
		label = scene.Point{X: cx, Y: cy + h/2 + reach + 4 + labelH/2}
	} else {
		label = scene.Point{X: cx + w/2 + reach + 4 + labelW/2, Y: cy}
	}
	p := new(scene.Path).MoveTo(at(0, -spread))
	ax, ay := at(reach, float64(-spread*2))
	bx, by := at(reach, float64(spread*2))
	ex, ey := at(tx, spread+ty)
	p.CubeTo(ax, ay, bx, by, ex, ey)
	e := EdgePath{Path: p}
	if sideways {
		e.SX, e.SY, e.SDX, e.SDY = cx-spread, cy+h/2, 0, -1
		e.EX, e.EY, e.EDX, e.EDY = cx+spread, cy+h/2, -spread/l, -reach/l
	} else {
		e.SX, e.SY, e.SDX, e.SDY = cx+w/2, cy-spread, -1, 0
		e.EX, e.EY, e.EDX, e.EDY = cx+w/2, cy+spread, -reach/l, -spread/l
	}
	return e, label
}

// LabelPatchRadius is how much the corners of an edge label's patch are rounded.
const LabelPatchRadius = 3

// LabelPatch is the patch of the background an edge's label centered on x, y is drawn on,
// so the line does not run through the text, as Pele's edgeLabelSvg draws it: the label's
// box and 4 more on the left and right, LabelPatchBox, its corners rounded by
// LabelPatchRadius. The part is a path, since the radius is not the theme's. (SVG's rect
// would round the corners of a patch less than 6 high elliptically, which no label is.)
func LabelPatch(l *text.Label, x, y float64) scene.Part {
	return scene.Part{Path: new(scene.Path).RoundRect(LabelPatchBox(l, x, y), LabelPatchRadius)}
}

// LabelPatchBox is the box of LabelPatch: the label's, centered on x, y, and 4 more on the
// left and right.
func LabelPatchBox(l *text.Label, x, y float64) scene.Rect {
	w, h := l.Width+8, l.Height
	return scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}
}
