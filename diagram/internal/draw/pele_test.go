package draw

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/text"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it.

type peleTips struct {
	D                string
	SX, SY, SDX, SDY float64
	EX, EY, EDX, EDY float64
}

type peleNode struct {
	X, Y, W, H float64
	IsGroup    bool
}

type peleDraw struct {
	Names []string
	Size  []struct {
		Shape    string
		TW, TH   float64
		W, H, DY float64
	}
	Inset []struct {
		Shape string
		W, H  float64
		Side  int
		T, V  float64
	}
	Span []struct {
		Shape    string
		W, H     float64
		Sideways bool
		V        float64
	}
	HasLabel []struct {
		Shape string
		V     bool
	}
	Ends []struct {
		Route []float64
		At    int
		Node  peleNode
		Shape string
		W, H  float64
		Side  int
		Out   []float64
	}
	Routes []struct {
		peleTips
		Route                []float64
		Curve                *string
		StartTrim, EndTrim   float64
		StartReach, EndReach *float64
	}
	Loops []struct {
		peleTips
		X, Y, W, H     float64
		K              int
		Sideways       bool
		Base, Trim     float64
		LabelW, LabelH float64
		LabelX, LabelY float64
	}
	Reach []struct {
		Adds [][3]json.RawMessage
		Of   [][3]float64
	}
	Titles []struct {
		Node   peleNode
		Routes [][]float64
		Marked [][2]json.RawMessage
		Tried  []struct {
			Width, Pad, X float64
			Probes        [][2]json.RawMessage
		}
	}
	Outlines []struct {
		Shape      string
		W, H, X, Y float64
		SVG        string
	}
	Markers []struct {
		Type         scene.MarkerType
		X, Y, DX, DY float64
		SVG          string
	}
	Patches []struct {
		Width, Height, X, Y float64
		SVG                 string
	}
	Struck []struct {
		Parents []int
		Index   int
		Tint    string
	}
	Hypot [][3]float64
}

func loadPele(t *testing.T) *peleDraw {
	t.Helper()
	var p peleDraw
	if err := fixture.Load("testdata/pele/draw.json", &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestShapesMatchPele(t *testing.T) {
	p := loadPele(t)
	names := make([]string, 0, len(shapes))
	for name := range shapes {
		names = append(names, name)
	}
	slices.Sort(names)
	want := slices.Sorted(slices.Values(p.Names))
	if !slices.Equal(names, want) {
		t.Errorf("shapes %v, Pele %v", names, want)
	}
	for _, c := range p.Size {
		if got := ShapeSize(c.Shape, c.TW, c.TH); got != (Size{c.W, c.H, c.DY}) {
			t.Errorf("ShapeSize(%q, %v, %v) = %v, Pele %v", c.Shape, c.TW, c.TH, got, Size{c.W, c.H, c.DY})
		}
	}
	for _, c := range p.Inset {
		if got := ShapeInset(c.Shape, c.W, c.H, c.Side, c.T); got != c.V {
			t.Errorf("ShapeInset(%q, %v, %v, %v, %v) = %v, Pele %v", c.Shape, c.W, c.H, c.Side, c.T, got, c.V)
		}
	}
	for _, c := range p.Span {
		if got := ShapeSpan(c.Shape, c.W, c.H, c.Sideways); got != c.V {
			t.Errorf("ShapeSpan(%q, %v, %v, %v) = %v, Pele %v", c.Shape, c.W, c.H, c.Sideways, got, c.V)
		}
	}
	for _, c := range p.HasLabel {
		if got := ShapeHasLabel(c.Shape); got != c.V {
			t.Errorf("ShapeHasLabel(%q) = %v", c.Shape, got)
		}
	}
	t.Logf("%d sizes, %d insets, %d spans, %d shapes", len(p.Size), len(p.Inset), len(p.Span), len(p.HasLabel))
}

func TestEndsMatchPele(t *testing.T) {
	p := loadPele(t)
	for _, c := range p.Ends {
		n := Node{c.Node.X, c.Node.Y, c.Node.W, c.Node.H, c.Node.IsGroup}
		if got := EndSide(c.Route, c.At, n); got != c.Side {
			t.Errorf("EndSide(%v, %d, %+v) = %d, Pele %d", c.Route, c.At, n, got, c.Side)
		}
		route := slices.Clone(c.Route)
		InsetRoute(route, c.At, c.Shape, c.W, c.H, n)
		if !slices.Equal(route, c.Out) {
			t.Errorf("InsetRoute(%v, %d, %q, %v, %v, %+v) = %v, Pele %v", c.Route, c.At, c.Shape, c.W, c.H, n, route, c.Out)
		}
	}
	t.Logf("%d ends", len(p.Ends))
}

func sameTips(t *testing.T, what string, e EdgePath, want peleTips) {
	t.Helper()
	got := peleTips{pelesvg.D(e.Path), e.SX, e.SY, e.SDX, e.SDY, e.EX, e.EY, e.EDX, e.EDY}
	if got != want {
		t.Errorf("%s\n got %+v\nPele %+v", what, got, want)
	}
}

func TestRoutePathMatchesPele(t *testing.T) {
	p := loadPele(t)
	curves := map[string]bool{}
	for _, c := range p.Routes {
		curve := ""
		if c.Curve != nil {
			curve = *c.Curve
		}
		curves[curve] = true
		startReach, endReach := OwnReach(c.StartTrim), OwnReach(c.EndTrim)
		if c.StartReach != nil {
			startReach = *c.StartReach
		}
		if c.EndReach != nil {
			endReach = *c.EndReach
		}
		e := RoutePath(c.Route, curve, c.StartTrim, c.EndTrim, startReach, endReach)
		sameTips(t, fmt.Sprintf("RoutePath(%v, %q, %v, %v, %v, %v)", c.Route, curve, c.StartTrim, c.EndTrim, startReach, endReach), e, c.peleTips)
	}
	t.Logf("%d routes, curves %v", len(p.Routes), curves)
}

func TestLoopPathMatchesPele(t *testing.T) {
	p := loadPele(t)
	for _, c := range p.Loops {
		e, at := LoopPath(c.X, c.Y, c.W, c.H, c.K, c.Sideways, c.Base, c.Trim, c.LabelW, c.LabelH)
		what := fmt.Sprintf("LoopPath(%v, %v, %v, %v, %v, %v, %v, %v, %v, %v)", c.X, c.Y, c.W, c.H, c.K, c.Sideways, c.Base, c.Trim, c.LabelW, c.LabelH)
		sameTips(t, what, e, c.peleTips)
		if at != (scene.Point{X: c.LabelX, Y: c.LabelY}) {
			t.Errorf("%s: label at %v, Pele %v, %v", what, at, c.LabelX, c.LabelY)
		}
	}
	t.Logf("%d loops", len(p.Loops))
}

func TestSideReachMatchesPele(t *testing.T) {
	p := loadPele(t)
	for _, c := range p.Reach {
		var r SideReach
		for _, a := range c.Adds {
			r.Add(decode[int](t, a[0]), decode[int](t, a[1]), decode[scene.MarkerType](t, a[2]))
		}
		for _, o := range c.Of {
			if got := r.Of(int(o[0]), int(o[1])); got != o[2] {
				t.Errorf("Of(%v, %v) = %v, Pele %v", o[0], o[1], got, o[2])
			}
		}
	}
}

func TestTitlesMatchPele(t *testing.T) {
	p := loadPele(t)
	for _, c := range p.Titles {
		n := Node{X: c.Node.X, Y: c.Node.Y, W: c.Node.W, H: c.Node.H}
		cr := Crossings{}
		for _, r := range c.Routes {
			MarkCrossings(cr, r)
		}
		if len(cr) != len(c.Marked) {
			t.Errorf("%d crossings, Pele %d", len(cr), len(c.Marked))
		}
		for _, m := range c.Marked {
			key, xs := decode[float64](t, m[0]), decode[[]float64](t, m[1])
			if !slices.Equal(cr[key], xs) {
				t.Errorf("crossings at %v: %v, Pele %v", key, cr[key], xs)
			}
		}
		for _, tr := range c.Tried {
			if got := ClusterTitleX(n, tr.Width, cr, tr.Pad); got != tr.X {
				t.Errorf("ClusterTitleX(%+v, %v, %v, %v) = %v, Pele %v", n, tr.Width, cr, tr.Pad, got, tr.X)
			}
			for _, pr := range tr.Probes {
				x, want := decode[float64](t, pr[0]), decode[bool](t, pr[1])
				if got := TitleCrossed(n, tr.Width, x, cr); got != want {
					t.Errorf("TitleCrossed(%+v, %v, %v, %v) = %v", n, tr.Width, x, cr, got)
				}
			}
		}
	}
	for _, c := range p.Struck {
		depth := StruckDepth(func(i int) int { return c.Parents[i] }, c.Index)
		if got := pelesvg.Num(StruckTint(depth)); got != c.Tint {
			t.Errorf("StruckTint of %v at %d = %v, Pele %v", c.Parents, c.Index, got, c.Tint)
		}
	}
	t.Logf("%d clusters, %d struck titles", len(p.Titles), len(p.Struck))
}

func TestHypotMatchesV8(t *testing.T) {
	for _, c := range loadPele(t).Hypot {
		if got := jsmath.Hypot(c[0], c[1]); got != c[2] {
			t.Errorf("jsmath.Hypot(%v, %v) = %v, V8 %v", c[0], c[1], got, c[2])
		}
	}
}

var (
	reTag  = regexp.MustCompile(`<(/?)(\w+)([^>]*?)(/?)>`)
	reAttr = regexp.MustCompile(`([\w-]+)="([^"]*)"`)
	reNum  = regexp.MustCompile(`-?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?`)
)

func attrs(s string) map[string]string {
	m := map[string]string{}
	for _, a := range reAttr.FindAllStringSubmatch(s, -1) {
		m[a[1]] = a[2]
	}
	return m
}

func numbers(t *testing.T, s string) []float64 {
	var out []float64
	for _, n := range reNum.FindAllString(s, -1) {
		v, err := strconv.ParseFloat(n, 64)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

func attrNum(t *testing.T, m map[string]string, name string) float64 {
	if s, ok := m[name]; ok {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("%s=%q", name, s)
		}
		return v
	}
	return 0
}

func TestMarkersMatchPele(t *testing.T) {
	p := loadPele(t)
	for _, c := range p.Markers {
		m := scene.Marker{Type: c.Type, At: scene.Point{X: c.X, Y: c.Y}, DX: c.DX, DY: c.DY}
		var path *scene.Path
		filled := false
		if parts := m.Outline(); len(parts) == 1 {
			path, filled = parts[0].Path, parts[0].Paint == scene.FillOnly
		} else if len(parts) > 1 {
			t.Errorf("%+v: %d parts", m, len(parts))
		}
		what := fmt.Sprintf("%+v", m)
		if c.SVG == "" {
			if path != nil {
				t.Errorf("%s: Pele draws none", what)
			}
			continue
		}
		if path == nil {
			t.Errorf("%s: none, Pele %s", what, c.SVG)
			continue
		}
		a := attrs(c.SVG)
		if wantFilled := a["fill"] == "C"; filled != wantFilled {
			t.Errorf("%s: filled %v, Pele %s", what, filled, c.SVG)
		}
		if strings.HasPrefix(c.SVG, "<circle") {
			b := path.Bounds()
			cx, cy := b.X+b.W/2, b.Y+b.H/2
			if math.Abs(cx-attrNum(t, a, "cx")) > 0.0051 || math.Abs(cy-attrNum(t, a, "cy")) > 0.0051 || math.Abs(b.W/2-attrNum(t, a, "r")) > 1e-9 {
				t.Errorf("%s: %v, Pele %s", what, b, c.SVG)
			}
			continue
		}
		if got := pelesvg.D(path); got != a["d"] {
			t.Errorf("%s: %s, Pele %s", what, got, a["d"])
		}
	}
	t.Logf("%d markers", len(p.Markers))
}

func TestLabelPatchMatchesPele(t *testing.T) {
	for _, c := range loadPele(t).Patches {
		part := LabelPatch(&text.Label{Width: c.Width, Height: c.Height}, c.X, c.Y)
		a := attrs(reTag.FindString(c.SVG[strings.Index(c.SVG, "<rect"):]))
		b := LabelPatchBox(&text.Label{Width: c.Width, Height: c.Height}, c.X, c.Y)
		got := []string{pelesvg.Num(b.X), pelesvg.Num(b.Y), pelesvg.Num(b.W), pelesvg.Num(b.H), "3"}
		want := []string{a["x"], a["y"], a["width"], a["height"], a["rx"]}
		if !slices.Equal(got, want) {
			t.Errorf("LabelPatch(%v, %v, %v, %v) = %v, Pele %v", c.Width, c.Height, c.X, c.Y, got, want)
		}
		if pb := part.Path.Bounds(); math.Abs(pb.X-b.X) > 1e-9 || math.Abs(pb.W-b.W) > 1e-9 || math.Abs(pb.H-b.H) > 1e-9 {
			t.Errorf("patch path %v, box %v", pb, b)
		}
	}
}

// The outlines are compared number for number where Pele draws with lines and curves: a
// polygon, a rect with square corners and a path with no arc in it must be the steps of the
// piece's path, and a rect with the theme's corners its box. The fixture has Pele's numbers
// in full; a path's are taken to hundredths, as Pele writes them, since the pieces take
// them as written. A circle, an ellipse, a rect with corners of its own and a path with an
// arc, which a scene.Path draws in curves that only come near, are compared as figures:
// Pele's markup and the pieces are each flattened to lines through points no more than
// about a unit apart, and each piece must have the paint of its element, the same bounding
// box, and every point near the other's lines.

type pt struct{ x, y float64 }

type figure struct {
	paint   scene.Painting
	rounded bool
	box     scene.Rect
	lines   [][]pt
	// steps is the element as a path, for one drawn with lines and curves; exact says the
	// steps, or the box of a rect with the theme's corners, are to be the piece's exactly.
	steps []pathStep
	exact bool
	rect  bool // the element is a rect, whose box a piece may have in place of a path
}

// A pathStep is one step of a path, as scene.Path.Each gives it.
type pathStep struct {
	op   scene.Op
	args []float64
}

// step is how far apart the points of a flattened curve are, at most.
const step = 1.0

func segments(n float64) int { return max(64, int(math.Ceil(n/step))) }

func dist(a, b pt) float64 { return math.Hypot(a.x-b.x, a.y-b.y) }

func quad(out []pt, a, c, b pt) []pt {
	n := segments(dist(a, c) + dist(c, b))
	for i := 1; i <= n; i++ {
		s := float64(i) / float64(n)
		u := 1 - s
		out = append(out, pt{u*u*a.x + 2*u*s*c.x + s*s*b.x, u*u*a.y + 2*u*s*c.y + s*s*b.y})
	}
	return out
}

func cube(out []pt, a, c1, c2, b pt) []pt {
	n := segments(dist(a, c1) + dist(c1, c2) + dist(c2, b))
	for i := 1; i <= n; i++ {
		s := float64(i) / float64(n)
		u := 1 - s
		out = append(out, pt{
			u*u*u*a.x + 3*u*u*s*c1.x + 3*u*s*s*c2.x + s*s*s*b.x,
			u*u*u*a.y + 3*u*u*s*c1.y + 3*u*s*s*c2.y + s*s*s*b.y,
		})
	}
	return out
}

// arc samples an SVG arc on the ellipse itself, by the conversion of SVG's implementation
// notes, F.6.5 and F.6.6, with no rotation.
func arc(out []pt, a pt, rx, ry float64, large, sweep bool, b pt) []pt {
	if a == b {
		return out
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return append(out, b)
	}
	x1p, y1p := (a.x-b.x)/2, (a.y-b.y)/2
	if l := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	sq := (rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p) / (rx*rx*y1p*y1p + ry*ry*x1p*x1p)
	co := math.Sqrt(max(0, sq))
	if large == sweep {
		co = -co
	}
	cxp, cyp := co*rx*y1p/ry, -co*ry*x1p/rx
	cx, cy := cxp+(a.x+b.x)/2, cyp+(a.y+b.y)/2
	t1 := math.Atan2((y1p-cyp)/ry, (x1p-cxp)/rx)
	t2 := math.Atan2((-y1p-cyp)/ry, (-x1p-cxp)/rx)
	dt := t2 - t1
	if sweep && dt < 0 {
		dt += 2 * math.Pi
	} else if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	}
	n := segments(math.Abs(dt) * max(rx, ry))
	for i := 1; i <= n; i++ {
		t := t1 + dt*float64(i)/float64(n)
		out = append(out, pt{cx + rx*math.Cos(t), cy + ry*math.Sin(t)})
	}
	return out
}

func ellipse(cx, cy, rx, ry float64) []pt {
	n := segments(2 * math.Pi * max(rx, ry))
	out := make([]pt, 0, n+1)
	for i := 0; i <= n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		out = append(out, pt{cx + rx*math.Cos(t), cy + ry*math.Sin(t)})
	}
	return out
}

// svgRect is a rect as SVG draws it, with rx and ry each made no more than half the side.
func svgRect(x, y, w, h, rx float64) []pt {
	rx, ry := min(rx, w/2), min(rx, h/2)
	if rx <= 0 {
		return []pt{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}, {x, y}}
	}
	out := []pt{{x + rx, y}}
	out = append(out, pt{x + w - rx, y})
	out = arc(out, pt{x + w - rx, y}, rx, ry, false, true, pt{x + w, y + ry})
	out = append(out, pt{x + w, y + h - ry})
	out = arc(out, pt{x + w, y + h - ry}, rx, ry, false, true, pt{x + w - rx, y + h})
	out = append(out, pt{x + rx, y + h})
	out = arc(out, pt{x + rx, y + h}, rx, ry, false, true, pt{x, y + h - ry})
	out = append(out, pt{x, y + ry})
	return arc(out, pt{x, y + ry}, rx, ry, false, true, pt{x + rx, y})
}

// svgPath flattens the d of a path, with the commands Pele's shapes use.
func svgPath(t *testing.T, d string, ox, oy float64) [][]pt {
	var lines [][]pt
	var line []pt
	var at, start, ctrl pt
	quadAt := false
	for _, c := range reCmd.FindAllString(d, -1) {
		op, n := c[0], hundredths(numbers(t, c[1:]))
		rel := op >= 'a'
		var dx, dy float64
		if rel {
			dx, dy = at.x, at.y
		}
		wasQuad := quadAt
		quadAt = false
		switch op | 0x20 {
		case 'm':
			if len(line) > 1 {
				lines = append(lines, line)
			}
			at = pt{n[0] + dx, n[1] + dy}
			start, line = at, []pt{at}
		case 'l':
			at = pt{n[0] + dx, n[1] + dy}
			line = append(line, at)
		case 'h':
			at = pt{n[0] + dx, at.y}
			line = append(line, at)
		case 'v':
			at = pt{at.x, n[0] + dy}
			line = append(line, at)
		case 'q':
			ctrl = pt{n[0] + dx, n[1] + dy}
			b := pt{n[2] + dx, n[3] + dy}
			line = quad(line, at, ctrl, b)
			at, quadAt = b, true
		case 't':
			c := at
			if wasQuad {
				c = pt{2*at.x - ctrl.x, 2*at.y - ctrl.y}
			}
			ctrl = c
			b := pt{n[0] + dx, n[1] + dy}
			line = quad(line, at, c, b)
			at, quadAt = b, true
		case 'c':
			b := pt{n[4] + dx, n[5] + dy}
			line = cube(line, at, pt{n[0] + dx, n[1] + dy}, pt{n[2] + dx, n[3] + dy}, b)
			at = b
		case 'a':
			if n[2] != 0 {
				t.Fatalf("a rotated arc in %q", d)
			}
			b := pt{n[5] + dx, n[6] + dy}
			line = arc(line, at, n[0], n[1], n[3] != 0, n[4] != 0, b)
			at = b
		case 'z':
			line = append(line, start)
			at = start
		}
	}
	if len(line) > 1 {
		lines = append(lines, line)
	}
	for _, l := range lines {
		for i := range l {
			l[i].x += ox
			l[i].y += oy
		}
	}
	return lines
}

var reCmd = regexp.MustCompile(`[MLHVCQTAZmlhvcqtaz][^MLHVCQTAZmlhvcqtaz]*`)

// hundredths is each of n to hundredths, as Pele's num writes it and SVG reads it back.
func hundredths(n []float64) []float64 {
	for i, v := range n {
		n[i] = jsmath.Round(float64(v*100)) / 100
	}
	return n
}

// pathSteps is the d of a path about o as the steps of a scene.Path: its numbers to
// hundredths, relative ones taken from where the path is and a T's control point the last
// one's reflected, in the shape's frame, as SVG reads them, and then moved by o. A path
// with an arc in it is nil.
func pathSteps(t *testing.T, d string, o pt) []pathStep {
	var out []pathStep
	add := func(op scene.Op, ps ...pt) {
		a := make([]float64, 0, 2*len(ps))
		for _, p := range ps {
			a = append(a, o.x+p.x, o.y+p.y)
		}
		out = append(out, pathStep{op, a})
	}
	var at, start, ctrl pt
	quadAt := false
	for _, c := range reCmd.FindAllString(d, -1) {
		op, n := c[0], hundredths(numbers(t, c[1:]))
		var r pt
		if op >= 'a' {
			r = at
		}
		if want := map[byte]int{'m': 2, 'l': 2, 'h': 1, 'v': 1, 'q': 4, 't': 2, 'c': 6, 'a': 7, 'z': 0}[op|0x20]; len(n) != want {
			t.Fatalf("%q in %q has %d numbers", c, d, len(n))
		}
		wasQuad := quadAt
		quadAt = false
		switch op | 0x20 {
		case 'm':
			at = pt{r.x + n[0], r.y + n[1]}
			start = at
			add(scene.MoveTo, at)
		case 'l':
			at = pt{r.x + n[0], r.y + n[1]}
			add(scene.LineTo, at)
		case 'h':
			at = pt{r.x + n[0], at.y}
			add(scene.LineTo, at)
		case 'v':
			at = pt{at.x, r.y + n[0]}
			add(scene.LineTo, at)
		case 'q':
			ctrl, at = pt{r.x + n[0], r.y + n[1]}, pt{r.x + n[2], r.y + n[3]}
			add(scene.QuadTo, ctrl, at)
			quadAt = true
		case 't':
			if wasQuad {
				ctrl = pt{2*at.x - ctrl.x, 2*at.y - ctrl.y}
			} else {
				ctrl = at
			}
			at = pt{r.x + n[0], r.y + n[1]}
			add(scene.QuadTo, ctrl, at)
			quadAt = true
		case 'c':
			at = pt{r.x + n[4], r.y + n[5]}
			add(scene.CubeTo, pt{r.x + n[0], r.y + n[1]}, pt{r.x + n[2], r.y + n[3]}, at)
		case 'a':
			return nil
		case 'z':
			add(scene.Close)
			at = start
		}
	}
	return out
}

// polygonSteps is a polygon about o as the steps of a scene.Path.
func polygonSteps(o pt, xy ...float64) []pathStep {
	out := []pathStep{{scene.MoveTo, []float64{o.x + xy[0], o.y + xy[1]}}}
	for i := 2; i+1 < len(xy); i += 2 {
		out = append(out, pathStep{scene.LineTo, []float64{o.x + xy[i], o.y + xy[i+1]}})
	}
	return append(out, pathStep{scene.Close, []float64{}})
}

// pieceSteps is the steps of a piece's path, or of its box where it has none.
func pieceSteps(p scene.Part) []pathStep {
	if p.Path == nil {
		b := p.Box
		return polygonSteps(pt{}, b.X, b.Y, b.X+b.W, b.Y, b.X+b.W, b.Y+b.H, b.X, b.Y+b.H)
	}
	var out []pathStep
	p.Path.Each(func(op scene.Op, a []float64) { out = append(out, pathStep{op, slices.Clone(a)}) })
	return out
}

func sameSteps(a, b []pathStep) bool {
	return slices.EqualFunc(a, b, func(x, y pathStep) bool { return x.op == y.op && slices.Equal(x.args, y.args) })
}

// peleFigures reads the elements of Pele's markup for a shape drawn about x, y.
func peleFigures(t *testing.T, svg string, x, y float64) []figure {
	var out []figure
	shift := []pt{{x, y}}
	for _, m := range reTag.FindAllStringSubmatch(svg, -1) {
		closing, name, a := m[1] == "/", m[2], attrs(m[3])
		at := shift[len(shift)-1]
		if name == "g" {
			if closing {
				shift = shift[:len(shift)-1]
			} else {
				n := numbers(t, a["transform"])
				shift = append(shift, pt{at.x + n[0], at.y + n[1]})
			}
			continue
		}
		f := figure{paint: scene.FillAndStroke}
		switch {
		case a["fill"] == "none" && a["stroke"] == "none":
			continue
		case a["fill"] == "none":
			f.paint = scene.StrokeOnly
		case a["fill"] == "var(--_l)":
			f.paint = scene.Solid
		case a["stroke"] == "none":
			f.paint = scene.FillOnly
		}
		switch name {
		case "rect":
			f.rect = true
			rx, w, h := a["rx"], attrNum(t, a, "width"), attrNum(t, a, "height")
			rx0 := at.x + attrNum(t, a, "x")
			ry0 := at.y + attrNum(t, a, "y")
			f.box = scene.Rect{X: rx0, Y: ry0, W: w, H: h}
			if strings.HasPrefix(rx, "var(--pele-radius") {
				f.rounded, f.exact = true, true
			} else if rx == "" {
				x0, y0 := attrNum(t, a, "x"), attrNum(t, a, "y")
				f.steps, f.exact = polygonSteps(at, x0, y0, x0+w, y0, x0+w, y0+h, x0, y0+h), true
			}
			r := 0.0
			if rx != "" && !f.rounded {
				r = attrNum(t, a, "rx")
			}
			f.lines = [][]pt{svgRect(rx0, ry0, w, h, r)}
		case "circle":
			r := attrNum(t, a, "r")
			f.lines = [][]pt{ellipse(at.x+attrNum(t, a, "cx"), at.y+attrNum(t, a, "cy"), r, r)}
		case "ellipse":
			f.lines = [][]pt{ellipse(at.x, at.y, attrNum(t, a, "rx"), attrNum(t, a, "ry"))}
		case "polygon":
			n := numbers(t, a["points"])
			var l []pt
			for i := 0; i+1 < len(n); i += 2 {
				l = append(l, pt{at.x + n[i], at.y + n[i+1]})
			}
			f.lines = [][]pt{append(l, l[0])}
			f.steps, f.exact = polygonSteps(at, n...), true
		case "path":
			f.lines = svgPath(t, a["d"], at.x, at.y)
			f.steps = pathSteps(t, a["d"], at)
			f.exact = f.steps != nil
		default:
			t.Fatalf("element %s in %s", name, svg)
		}
		out = append(out, f)
	}
	return out
}

// pieceFigure flattens a piece of an outline.
func pieceFigure(p scene.Part) figure {
	f := figure{paint: p.Paint, rounded: p.Rounded, box: p.Box}
	if p.Path == nil {
		b := p.Box
		f.lines = [][]pt{{{b.X, b.Y}, {b.X + b.W, b.Y}, {b.X + b.W, b.Y + b.H}, {b.X, b.Y + b.H}, {b.X, b.Y}}}
		return f
	}
	var line []pt
	var at, start pt
	p.Path.Each(func(op scene.Op, a []float64) {
		switch op {
		case scene.MoveTo:
			if len(line) > 1 {
				f.lines = append(f.lines, line)
			}
			at = pt{a[0], a[1]}
			start, line = at, []pt{at}
		case scene.LineTo:
			at = pt{a[0], a[1]}
			line = append(line, at)
		case scene.QuadTo:
			b := pt{a[2], a[3]}
			line = quad(line, at, pt{a[0], a[1]}, b)
			at = b
		case scene.CubeTo:
			b := pt{a[4], a[5]}
			line = cube(line, at, pt{a[0], a[1]}, pt{a[2], a[3]}, b)
			at = b
		case scene.Close:
			line = append(line, start)
			at = start
		}
	})
	if len(line) > 1 {
		f.lines = append(f.lines, line)
	}
	return f
}

func bounds(lines [][]pt) scene.Rect {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, l := range lines {
		for _, p := range l {
			x0, y0, x1, y1 = min(x0, p.x), min(y0, p.y), max(x1, p.x), max(y1, p.y)
		}
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// farthest is the farthest any point of a lies from the lines of b.
func farthest(a, b [][]pt) float64 {
	worst := 0.0
	for _, la := range a {
		for _, p := range la {
			best := math.Inf(1)
			for _, lb := range b {
				for i := 1; i < len(lb) && best > 0; i++ {
					best = min(best, toSegment(p, lb[i-1], lb[i]))
				}
			}
			worst = max(worst, best)
		}
	}
	return worst
}

func toSegment(p, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l := dx*dx + dy*dy
	s := 0.0
	if l > 0 {
		s = max(0, min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/l))
	}
	return math.Hypot(p.x-a.x-s*dx, p.y-a.y-s*dy)
}

func TestOutlinesMatchPele(t *testing.T) {
	p := loadPele(t)
	// Four cubic curves stand for a circle to within 0.027% of its radius, and the numbers
	// of Pele's markup for a circle, an ellipse or a rect are in full, where it writes them
	// to hundredths.
	shapesSeen, exact, all := map[string]bool{}, 0, 0
	for _, c := range p.Outlines {
		shapesSeen[c.Shape] = true
		tol := 0.05 + 0.00028*max(c.W, c.H)/2
		what := fmt.Sprintf("Outline(%q, %v, %v, %v, %v)", c.Shape, c.X, c.Y, c.W, c.H)
		want := peleFigures(t, c.SVG, c.X, c.Y)
		pieces := Outline(c.Shape, c.X, c.Y, c.W, c.H)
		if len(pieces) != len(want) {
			t.Errorf("%s: %d pieces, Pele %d: %s", what, len(pieces), len(want), c.SVG)
			continue
		}
		for i, piece := range pieces {
			got, w := pieceFigure(piece), want[i]
			if got.paint != w.paint || got.rounded != w.rounded {
				t.Errorf("%s: piece %d paint %v rounded %v, Pele paint %v rounded %v", what, i, got.paint, got.rounded, w.paint, w.rounded)
			}
			all++
			if w.exact {
				exact++
			}
			switch {
			case w.rounded:
				if piece.Path != nil || piece.Box != w.box {
					t.Errorf("%s: piece %d box %v, Pele %v", what, i, piece.Box, w.box)
				}
				continue
			case w.exact && w.rect && piece.Path == nil:
				if piece.Box != w.box {
					t.Errorf("%s: piece %d box %v, Pele %v", what, i, piece.Box, w.box)
				}
				continue
			case w.exact:
				if got := pieceSteps(piece); !sameSteps(got, w.steps) {
					t.Errorf("%s: piece %d is\n%v,\nPele\n%v: %s", what, i, got, w.steps, c.SVG)
				}
				continue
			}
			gb, wb := bounds(got.lines), bounds(w.lines)
			if math.Abs(gb.X-wb.X) > tol || math.Abs(gb.Y-wb.Y) > tol || math.Abs(gb.X+gb.W-wb.X-wb.W) > tol || math.Abs(gb.Y+gb.H-wb.Y-wb.H) > tol {
				t.Errorf("%s: piece %d bounds %v, Pele %v: %s", what, i, gb, wb, c.SVG)
			}
			if d := max(farthest(got.lines, w.lines), farthest(w.lines, got.lines)); d > tol {
				t.Errorf("%s: piece %d is %v from Pele's: %s", what, i, d, c.SVG)
			}
		}
	}
	for name := range shapes {
		if !shapesSeen[name] {
			t.Errorf("no outline of %q", name)
		}
	}
	t.Logf("%d outlines, %d of their %d pieces compared number for number", len(p.Outlines), exact, all)
}
