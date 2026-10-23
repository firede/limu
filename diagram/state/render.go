package state

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/diagram/internal/draw"
	"github.com/firede/limu/diagram/internal/graphs"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing of a state diagram, as Pele's render.ts has it.
const (
	nodeSep   = 40
	edgeSep   = 16
	rankSep   = 44
	groupPad  = 16
	regionPad = 10
	padX      = 16
	padY      = 10
	loop      = 26
	fold      = 8
	noteWrap  = 320
	corner    = 10
	bar       = 8
)

// kind is what a node is drawn as, Pele's K.
type kind int

const (
	kState kind = iota
	kDescribed
	kStart
	kEnd
	kChoice
	kFork
	kJoin
	kNote
	kComposite
	kRegion
	kHidden
)

var kinds = map[string]kind{
	"rectWithTitle": kDescribed,
	"stateStart":    kStart,
	"stateEnd":      kEnd,
	"choice":        kChoice,
	"fork":          kFork,
	"join":          kJoin,
	"note":          kNote,
	"divider":       kHidden,
}

// The roles of the items of a state diagram.
var roles = [...]scene.Role{
	kState:     RoleState,
	kDescribed: RoleState,
	kStart:     RoleStart,
	kEnd:       RoleEnd,
	kChoice:    RoleChoice,
	kFork:      RoleFork,
	kJoin:      RoleJoin,
	kNote:      RoleNote,
	kComposite: RoleComposite,
}

// A view is a node as it is drawn, Pele's View. node is nil for the unseen helpers the
// layout needs.
type view struct {
	node     *GraphNode
	kind     kind
	author   css.Author
	hasStyle bool
	label    text.Label
	lines    []text.Label
	w, h     float64
	titleH   float64
}

// Layout lays out a parsed state diagram as a scene, as Pele's renderState draws it: in its
// direction, and, for one that runs across and does not fit o.MaxWidth, down instead where
// that is narrower.
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if !common.RunsAcross(db.Direction()) {
		return lay(db, config, o, false)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(db, config, o, down) })
}

func vertical(dir graphs.Dir) bool { return dir == graphs.TB || dir == graphs.BT }

// numberOption is a number of the state diagram's config, if it is one above 0, at most 2000,
// as Pele's numberOption reads it.
func numberOption(config diagram.Config, key string, fallback float64) float64 {
	section, _ := config["state"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// unquote takes off the quotes Mermaid keeps around a click statement's strings, as Pele's
// unquote does.
func unquote(s string) string {
	return strings.TrimRight(strings.TrimLeft(s, `"`), `"`)
}

func clamp(value, lo, hi float64) float64 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

// group is a cnode for a group, as Pele's group makes.
func group(dir graphs.Dir, padX, padTop, padBottom float64) graphs.CNode {
	c := graphs.NewCNode(0, 0, -1)
	c.IsGroup = true
	c.Dir = dir
	c.PadX = padX
	c.PadTop = padTop
	c.PadBottom = padBottom
	return c
}

// classNames is a node's classes without the ones Mermaid gives every state, cluster and
// note, as Pele cleans them for its class attribute.
func classNames(raw string) []string {
	var kept []string
	for _, name := range strings.Split(raw, " ") {
		switch name {
		case "statediagram-state", "statediagram-cluster-alt", "statediagram-cluster", "statediagram-note":
		default:
			kept = append(kept, name)
		}
	}
	return common.ClassNames(strings.Join(kept, " "))
}

// lay lays the diagram out once, as Pele's draw does. turned lays out a diagram that runs
// across as one that runs down.
func lay(db *DB, config diagram.Config, o *layout.Options, turned bool) *scene.Scene {
	graph := db.Graph
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	wrapWidth := numberOption(config, "wrappingWidth", 200)
	pad := o.Pad()
	rootDir := graphs.Direction(db.Direction())
	if turned {
		rootDir = graphs.TB
	}
	labelOf := func(raw string, fontSize, wrap float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: fontSize, MaxWidth: wrap, Face: face, Markdown: true, Icons: o.Icons})
	}

	index := map[string]int{}
	var views []view
	var cnodes []graphs.CNode
	add := func(node *GraphNode, k kind, c *graphs.CNode) int {
		views = append(views, view{node: node, kind: k})
		if c != nil {
			cnodes = append(cnodes, *c)
		}
		return len(views) - 1
	}
	for _, node := range graph.Nodes {
		if node.Shape == "noteGroup" {
			continue
		}
		k, ok := kinds[node.Shape]
		if !ok {
			k = kState
		}
		if node.IsGroup {
			k = kComposite
			if node.Shape == "divider" {
				k = kRegion
			}
		}
		index[node.ID] = add(node, k, nil)
	}
	states := len(views)

	logical := make([]int, states)
	for i := range states {
		logical[i] = -1
		if n := views[i].node; n.HasParent {
			if parent, ok := index[n.Parent]; ok && parent != i && views[parent].node.IsGroup {
				logical[i] = parent
			}
		}
	}
	// Composites that are redeclared inside each other form a ring with no way in from the
	// root. One member of each ring moves to the root, which leaves a tree.
	{
		kids := make([][]int, states)
		var queue []int
		placed := make([]bool, states)
		for i := range states {
			if logical[i] == -1 {
				placed[i] = true
				queue = append(queue, i)
			} else {
				kids[logical[i]] = append(kids[logical[i]], i)
			}
		}
		done := 0
		settle := func() {
			for ; done < len(queue); done++ {
				for _, kid := range kids[queue[done]] {
					if placed[kid] {
						continue
					}
					placed[kid] = true
					queue = append(queue, kid)
				}
			}
		}
		settle()
		seen := make([]int, states)
		for i := range seen {
			seen[i] = -1
		}
		for i := range states {
			if placed[i] {
				continue
			}
			ring := i
			for seen[ring] != i {
				seen[ring] = i
				ring = logical[ring]
			}
			logical[ring] = -1
			placed[ring] = true
			queue = append(queue, ring)
			settle()
		}
	}
	owners := make([]int, states)
	for i := range states {
		owners[i] = -1
		n := views[i].node
		if n.NoteFor == "" {
			continue
		}
		owner, ok := index[n.NoteFor]
		if !ok {
			continue
		}
		owners[i] = owner
		logical[i] = logical[owner]
	}

	// As in Mermaid, a composite that sets no direction runs top to bottom whatever surrounds
	// it.
	innerDir := func(composite int) graphs.Dir { return graphs.Direction(views[composite].node.Dir) }
	// Whether the flow around a state runs top to bottom or bottom to top.
	across := func(i int) bool {
		if logical[i] == -1 {
			return vertical(rootDir)
		}
		return vertical(innerDir(logical[i]))
	}

	degree := make([]int, states)
	noted := make([]bool, states)
	// When each state first takes part in a transition. Layout follows this order rather than
	// the order of declaration, so a state declared up front does not become the top of the
	// flow.
	appear := make([]float64, states)
	for i := range appear {
		appear[i] = math.Inf(1)
	}
	appeared := 0.0
	counts := make([]int, 2*states)
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		if edge.Pattern == "dashed" {
			if views[src].kind == kNote {
				noted[dst] = true
			} else {
				noted[src] = true
			}
			continue
		}
		counts[2*src]++
		degree[src] = max(degree[src], counts[2*src])
		counts[2*dst+1]++
		degree[dst] = max(degree[dst], counts[2*dst+1])
		if math.IsInf(appear[src], 1) {
			appear[src] = appeared
			appeared++
		}
		if math.IsInf(appear[dst], 1) {
			appear[dst] = appeared
			appeared++
		}
	}

	for i := range states {
		v := &views[i]
		node := v.node
		decls := node.CSSCompiledStyles
		if len(node.CSSStyles) > 0 {
			decls = append(append([]string(nil), node.CSSCompiledStyles...), node.CSSStyles...)
		}
		author := css.Declarations(decls)
		var face text.Face
		if author.Bold {
			face |= text.Bold
		}
		if author.Italic {
			face |= text.Italic
		}
		fontSize := size
		if author.FontSize > 0 {
			fontSize = author.FontSize
		}
		v.author, v.hasStyle = author, true
		var c graphs.CNode
		made := false
		switch v.kind {
		case kComposite:
			v.label = labelOf(node.Label, fontSize, 4000, face)
			if v.label.Height > 0 {
				v.titleH = v.label.Height + 12
			}
			c = group(innerDir(i), groupPad, v.titleH+groupPad, groupPad)
			c.MinW = v.label.Width + 2*groupPad
			made = true
		case kRegion:
			c = group(innerDir(i), regionPad, regionPad, regionPad)
			made = true
		case kStart:
			v.w, v.h = 14, 14
		case kEnd:
			v.w, v.h = 18, 18
		case kChoice:
			v.w, v.h = 30, 30
		case kFork, kJoin:
			length := clamp(float64(40*degree[i]+16), 72, 320)
			if across(i) {
				v.w, v.h = length, bar
			} else {
				v.w, v.h = bar, length
			}
		case kHidden:
		case kNote:
			v.label = labelOf(node.Label, small, max(wrapWidth, noteWrap), 0)
			v.w = v.label.Width + 24
			v.h = v.label.Height + 16
		default:
			v.label = labelOf(node.Label, fontSize, wrapWidth, face)
			width := v.label.Width
			height := v.label.Height + 2*padY
			if v.kind == kDescribed {
				height += 14
				for _, line := range node.Description {
					if line == "" {
						line = " "
					}
					label := labelOf(line, small, wrapWidth, face)
					v.lines = append(v.lines, label)
					if label.Width > width {
						width = label.Width
					}
					height += label.Height
				}
			}
			v.w = max(width+2*padX, 56)
			v.h = height
		}
		if !made {
			c = graphs.NewCNode(v.w, v.h, -1)
		}
		// The transitions on one side of a state's box spread along it, clear of its corners.
		if v.kind == kState || v.kind == kDescribed {
			if across(i) {
				c.Span = v.w - 24
			} else {
				c.Span = v.h - 24
			}
		}
		cnodes = append(cnodes, c)
	}

	var cedges []graphs.CEdge
	var edgeLabels []text.Label
	var drawn []*GraphEdge
	// The transitions each state makes to itself: how many, their widest label, and their
	// labels' height.
	loopCount := make([]int, states)
	loopWidth := make([]float64, states)
	loopHeight := make([]float64, states)
	// A transition between a composite and a state inside it runs between that state and a
	// point just inside the composite, and is then drawn on to the composite's border.
	anchors := map[int]int{}
	encloses := func(composite, item int) bool {
		if !cnodes[composite].IsGroup {
			return false
		}
		for p := logical[item]; p != -1; p = logical[p] {
			if p == composite {
				return true
			}
		}
		return false
	}
	anchor := func(composite int, first bool) int {
		anchors[len(cedges)] = composite
		if first {
			appear = append(appear, -1)
		} else {
			appear = append(appear, math.Inf(1))
		}
		c := graphs.NewCNode(0, 0, composite)
		return add(nil, kHidden, &c)
	}
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		label := labelOf(edge.Label, small, wrapWidth, 0)
		labelW := 0.0
		if label.Width > 0 {
			labelW = label.Width + 12
		}
		switch {
		case src == dst:
			loopCount[src]++
			loopWidth[src] = max(loopWidth[src], labelW)
			loopHeight[src] += label.Height
		case encloses(src, dst):
			src = anchor(src, true)
		case encloses(dst, src):
			dst = anchor(dst, false)
		}
		drawn = append(drawn, edge)
		edgeLabels = append(edgeLabels, label)
		ce := graphs.CEdge{Src: src, Dst: dst, Minlen: 1, LabelW: labelW}
		if label.Height > 0 {
			ce.LabelH = label.Height + 4
		}
		cedges = append(cedges, ce)
	}

	// A state with notes, or a composite with self-transitions, sits in an unseen group of its
	// own: the notes go beside the state, across the flow, without disturbing its transitions.
	for i := range states {
		if owners[i] != -1 {
			continue
		}
		// Loops hang off the right, or below when the flow is horizontal, with their labels
		// beyond them.
		room := 0.0
		if loopCount[i] > 0 {
			reach := float64(loop + (loopCount[i]-1)*8)
			if across(i) {
				room = reach + loopWidth[i]
			} else {
				room = float64(reach*0.75) + 4 + loopHeight[i]
			}
			if item := &cnodes[i]; !item.IsGroup {
				if across(i) {
					item.W += float64(2 * room)
					item.H = max(item.H, loopHeight[i])
				} else {
					item.H += float64(2 * room)
					item.W = max(item.W, loopWidth[i])
				}
				room = 0
			}
		}
		if !noted[i] && room == 0 {
			cnodes[i].Parent = logical[i]
			continue
		}
		var c graphs.CNode
		if across(i) {
			c = group(graphs.LR, room, 0, 0)
		} else {
			c = group(graphs.TB, 0, room, room)
		}
		c.Parent = logical[i]
		appear = append(appear, appear[i])
		parent := add(nil, kHidden, &c)
		cnodes[i].Parent = parent
	}
	for i := range states {
		if owners[i] != -1 {
			cnodes[i].Parent = cnodes[owners[i]].Parent
		}
	}

	{
		// Numbers the tree so that, at every level, initial states come first and the rest
		// follow in the order the transitions introduce them.
		members := make([][]int, len(cnodes))
		var top []int
		for i := range cnodes {
			if p := cnodes[i].Parent; p >= 0 {
				members[p] = append(members[p], i)
			} else {
				top = append(top, i)
			}
		}
		initial := func(i int) int {
			if views[i].kind == kStart {
				return 0
			}
			return 1
		}
		byFlow := func(a, b int) int {
			if d := initial(a) - initial(b); d != 0 {
				return d
			}
			if appear[a] != appear[b] {
				if appear[a] < appear[b] {
					return -1
				}
				return 1
			}
			return a - b
		}
		seq := 0.0
		slices.SortStableFunc(top, byFlow)
		slices.Reverse(top)
		pending := top
		for len(pending) > 0 {
			i := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			cnodes[i].Seq = seq
			seq++
			inside := members[i]
			slices.SortStableFunc(inside, byFlow)
			for k := len(inside) - 1; k >= 0; k-- {
				pending = append(pending, inside[k])
			}
		}
	}

	// State diagrams follow the flowchart curve, as Pele's draw does, unless the host asks for
	// transitions at right angles, or straight.
	flow, _ := config["flowchart"].(diagram.Config)
	curve, _ := flow["curve"].(string)
	if routed := common.RoutingCurve(o.Routing); routed != "" {
		curve = routed
	}
	result := graphs.Compound(cnodes, cedges, rootDir, graphs.LayeredOptions{
		NodeSep:  numberOption(config, "nodeSpacing", nodeSep),
		EdgeSep:  edgeSep,
		RankSep:  numberOption(config, "rankSpacing", rankSep),
		PortSep:  20,
		Tracks:   curve == "step" || curve == "stepBefore" || curve == "stepAfter",
		Straight: curve == "linear",
		HeadRoom: scene.MarkerType("arrow_point").Trim(),
	})

	width := math.Ceil(result.Width + 2*pad)
	height := math.Ceil(result.Height + 2*pad)
	links := !o.NoLinks

	// Route points by rounded y, to find the edges that cross a composite's title bar.
	crossings := map[float64][]float64{}
	for _, e := range cedges {
		route := e.Route
		for k := 3; k < len(route)-3; k += 3 {
			key := jsmath.Round(route[k+1])
			crossings[key] = append(crossings[key], route[k])
		}
	}

	children := make([][]int, len(cnodes))
	var order []int
	for i := range cnodes {
		if p := cnodes[i].Parent; p >= 0 {
			children[p] = append(children[p], i)
		} else {
			order = append(order, i)
		}
	}
	roots := slices.Clone(order)
	for q := 0; q < len(order); q++ {
		order = append(order, children[order[q]]...)
	}

	var clusters, dividers, nodes, notes, edges, labels []scene.Item
	dividerStyle := style.Style{Stroke: style.Border, Fill: style.None, Dash: style.Dashes(4, 4), Radius: style.N(corner)}
	// divide draws the dashed lines between the regions among a composite's members.
	divide := func(items []int, left, top, right, bottom float64) {
		var regions []int
		for _, i := range items {
			if views[i].kind == kRegion {
				regions = append(regions, i)
			}
		}
		if len(regions) < 2 {
			return
		}
		lo := func(i int, x bool) float64 {
			if x {
				return cnodes[i].X - cnodes[i].W/2
			}
			return cnodes[i].Y - cnodes[i].H/2
		}
		hi := func(i int, x bool) float64 {
			if x {
				return cnodes[i].X + cnodes[i].W/2
			}
			return cnodes[i].Y + cnodes[i].H/2
		}
		for _, x := range [2]bool{true, false} {
			sorted := slices.Clone(regions)
			slices.SortStableFunc(sorted, func(a, b int) int {
				if d := lo(a, x) - lo(b, x); d != 0 {
					if d < 0 {
						return -1
					}
					return 1
				}
				return a - b
			})
			apart := true
			for k := 1; k < len(sorted); k++ {
				if hi(sorted[k-1], x) > lo(sorted[k], x)+0.5 {
					apart = false
				}
			}
			if !apart {
				continue
			}
			for k := 1; k < len(sorted); k++ {
				at := (hi(sorted[k-1], x)+lo(sorted[k], x))/2 + pad
				p := new(scene.Path)
				if x {
					p.MoveTo(at, top).LineTo(at, bottom)
				} else {
					p.MoveTo(left, at).LineTo(right, at)
				}
				dividers = append(dividers, scene.Item{
					Kind: scene.Mark, Role: RoleDivider, Box: p.Bounds(),
					Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}}, Style: dividerStyle,
				})
			}
			return
		}
		// Regions that sit neither side by side nor in a stack each get an outline.
		for _, i := range regions {
			c := &cnodes[i]
			box := scene.Rect{X: c.X - c.W/2 + pad, Y: c.Y - c.H/2 + pad, W: c.W, H: c.H}
			dividers = append(dividers, scene.Item{
				Kind: scene.Mark, Role: RoleDivider, Box: box,
				Parts: []scene.Part{{Box: box, Rounded: true, Paint: scene.StrokeOnly}}, Style: dividerStyle,
			})
		}
	}
	divide(roots, pad/2, pad/2, width-pad/2, height-pad/2)

	for _, i := range order {
		v := &views[i]
		node := v.node
		if node == nil || !v.hasStyle || v.kind >= kRegion {
			continue
		}
		c := &cnodes[i]
		x := c.X + pad
		y := c.Y + pad
		w, h := v.w, v.h
		author := v.author.Style
		it := scene.Item{
			Kind: scene.Node, Role: roles[v.kind], Shape: scene.Shape(node.Shape), ID: node.ID,
			Class: classNames(node.CSSClasses), Box: scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h},
			Author: author,
		}
		var extra []scene.Item

		switch v.kind {
		case kComposite:
			left := x - c.W/2
			top := y - c.H/2
			box := scene.Rect{X: left, Y: top, W: c.W, H: c.H}
			it.Kind, it.Box = scene.Group, box
			it.Parts = []scene.Part{{Box: box, Rounded: true}}
			it.Style = style.Style{Fill: style.SurfaceAlt, FillOpacity: style.N(0.5), Stroke: style.Border, Text: style.Fg, Radius: style.N(corner)}
			if v.titleH > 0 {
				rule := new(scene.Path).MoveTo(left, top+v.titleH).LineTo(left+c.W, top+v.titleH)
				it.Parts = append(it.Parts, scene.Part{Path: rule, Paint: scene.StrokeOnly})
				if !v.label.Empty() {
					it.Label = &v.label
					it.At = scene.Point{X: titleX(c, v.label.Width, crossings) + pad, Y: top + v.titleH/2}
				}
			}
			divide(children[i], left, top+v.titleH, left+c.W, top+c.H)
		case kStart:
			it.Parts = []scene.Part{{Path: new(scene.Path).Ellipse(x, y, 7, 7)}}
			it.Style = style.Style{Fill: style.Line, Stroke: style.None}
		case kEnd:
			it.Parts = []scene.Part{
				{Path: new(scene.Path).Ellipse(x, y, 8.5, 8.5), Paint: scene.Hollow},
				{Path: new(scene.Path).Ellipse(x, y, 5, 5), Paint: scene.FillOnly},
			}
			it.Style = style.Style{Fill: style.Line, Stroke: style.Line}
		case kChoice:
			it.Parts = []scene.Part{{Path: new(scene.Path).Polygon(x, y-h/2, x+w/2, y, x, y+h/2, x-w/2, y)}}
			it.Style = style.Style{Fill: style.Surface, Stroke: style.Border}
		case kFork, kJoin:
			it.Parts = []scene.Part{{Box: it.Box, Rounded: true}}
			it.Style = style.Style{Fill: style.Line, Stroke: style.None, Radius: style.N(2)}
		case kNote:
			r, b := w/2, h/2
			outline := new(scene.Path).MoveTo(x-r, y-b).LineTo(x+r-fold, y-b).LineTo(x+r, y+fold-b).
				LineTo(x+r, y+b).LineTo(x-r, y+b).Close()
			dogEar := new(scene.Path).MoveTo(x+r-fold, y-b).LineTo(x+r-fold, y+fold-b).LineTo(x+r, y+fold-b)
			it.Parts = []scene.Part{{Path: outline}, {Path: dogEar, Paint: scene.StrokeOnly}}
			it.Style = style.Style{Fill: style.Bg, Stroke: style.Border, Text: style.Muted}
			// A note takes no style of its author's.
			it.Author = style.Style{}
			it.Label, it.At = &v.label, scene.Point{X: x, Y: y}
		default:
			it.Parts = []scene.Part{{Box: it.Box, Rounded: true}}
			it.Style = style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg, Radius: style.N(min(corner, h/2))}
			if v.kind == kState {
				it.Label, it.At = &v.label, scene.Point{X: x, Y: y}
				break
			}
			at := padY - h/2 + v.label.Height
			rule := new(scene.Path).MoveTo(x-w/2, y+at+6).LineTo(x+w/2, y+at+6)
			it.Parts = append(it.Parts, scene.Part{Path: rule, Paint: scene.StrokeOnly})
			it.Label, it.At = &v.label, scene.Point{X: x, Y: y + (at - v.label.Height/2)}
			at += 14
			for k := range v.lines {
				line := &v.lines[k]
				extra = append(extra, scene.Item{
					Kind: scene.Text, Role: RoleDescription, ID: node.ID, Label: line,
					At:     scene.Point{X: x, Y: y + (at + line.Height/2)},
					Style:  style.Style{Text: style.Fg},
					Author: style.Style{Text: author.Text},
				})
				at += line.Height
			}
		}

		if i, ok := db.links[node.ID]; ok {
			link := db.Links[i]
			// Entity codes are resolved first, so the URL is checked in the form a browser
			// would follow.
			it.Tooltip = text.DecodeEntities(unquote(link.Tooltip))
			if href, ok := url.LinkURL(url.SafeURL(unquote(link.URL)), links, o.LinkSchemes); ok {
				it.Link = href
			}
		}
		switch v.kind {
		case kComposite:
			clusters = append(clusters, it)
		case kNote:
			notes = append(notes, it)
		default:
			nodes = append(nodes, it)
			nodes = append(nodes, extra...)
		}
	}

	node := func(i int) draw.Node {
		c := &cnodes[i]
		return draw.Node{X: c.X, Y: c.Y, W: c.W, H: c.H, Group: c.IsGroup}
	}
	// attach moves a route end from the layout box onto the drawn shape, and spreads the ends
	// along a bar.
	attach := func(route []float64, at, i int) {
		c := &cnodes[i]
		if c.IsGroup {
			return
		}
		v := &views[i]
		isBar := v.kind == kFork || v.kind == kJoin
		other := at - 3
		if at == 0 {
			other = 3
		}
		// 0 is the x axis and 1 the y axis; the end sits on a side that faces along axis.
		for _, axis := range [2]int{1, 0} {
			extent, drawnExtent, breadth := c.W, v.w, v.h
			center, cross := c.X+pad, c.Y+pad
			if axis == 1 {
				extent, drawnExtent, breadth = c.H, v.h, v.w
				center, cross = c.Y+pad, c.X+pad
			}
			delta := route[at+axis] - center
			if math.Abs(math.Abs(delta)-extent/2) >= 0.5 {
				continue
			}
			route[at+axis] -= float64(jsmath.Sign(delta)*(extent-drawnExtent)) / 2
			if isBar && breadth > drawnExtent {
				route[at+1-axis] = clamp(route[other+1-axis], cross-breadth/2+6, cross+breadth/2-6)
			}
			return
		}
	}

	var reach draw.SideReach
	for i, ce := range cedges {
		if ce.Src == ce.Dst || len(ce.Route) < 6 || drawn[i].Pattern == "dashed" {
			continue
		}
		reach.Add(ce.Dst, draw.EndSide(ce.Route, len(ce.Route)-3, node(ce.Dst)), "arrow_point")
	}

	loopSeen := make([]int, states)
	loopText := make([]float64, states)
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		label := &edgeLabels[i]
		note := edge.Pattern == "dashed"
		var endType scene.MarkerType
		if !note {
			endType = "arrow_point"
		}
		var path draw.EdgePath
		var route []float64
		if ce.Src == ce.Dst {
			a := &cnodes[ce.Src]
			v := &views[ce.Src]
			k := loopSeen[ce.Src]
			loopSeen[ce.Src]++
			// The loop hangs off a side the flow does not use: the right, or below when the
			// flow is horizontal.
			side := across(ce.Src)
			w, h := v.w, v.h
			if a.IsGroup {
				w, h = a.W, a.H
			}
			out, along := h/2, w/2
			if side {
				out, along = w/2, h/2
			}
			spread := max(min(along-4, float64(8+k*6)), 2)
			reach := float64(loop + k*8)
			trim := endType.Trim() / jsmath.Or1(jsmath.Hypot(reach, spread))
			// Offsets are given for a loop on the right and turned a quarter for one below.
			px := func(dx, dy float64) float64 {
				if side {
					return a.X + pad + (out + float64(dx))
				}
				return a.X + pad + -dy
			}
			py := func(dx, dy float64) float64 {
				if side {
					return a.Y + pad + dy
				}
				return a.Y + pad + (out + float64(dx))
			}
			p := new(scene.Path).MoveTo(px(0, -spread), py(0, -spread)).CubeTo(
				px(reach, -2*spread), py(reach, -2*spread),
				px(reach, 2*spread), py(reach, 2*spread),
				px(reach*trim, spread+float64(spread*trim)), py(reach*trim, spread+float64(spread*trim)))
			path = draw.EdgePath{Path: p, EX: px(0, spread), EY: py(0, spread),
				EDX: px(-reach, -spread) - px(0, 0), EDY: py(-reach, -spread) - py(0, 0)}
			length := jsmath.Or1(jsmath.Hypot(path.EDX, path.EDY))
			path.EDX /= length
			path.EDY /= length
			// Labels stack beyond the outermost loop, in the order of their transitions.
			furthest := float64(loop + (loopCount[ce.Src]-1)*8)
			below := loopText[ce.Src] + label.Height/2
			loopText[ce.Src] += label.Height
			if side {
				ce.LabelX = a.X + out + furthest + 4 + ce.LabelW/2
				ce.LabelY = a.Y - loopHeight[ce.Src]/2 + below
			} else {
				ce.LabelX = a.X
				ce.LabelY = a.Y + out + float64(furthest*0.75) + 4 + below
			}
		} else {
			route = slices.Clone(ce.Route)
			leaves := reach.Of(ce.Src, draw.EndSide(route, 0, node(ce.Src)))
			arrives := reach.Of(ce.Dst, draw.EndSide(route, len(route)-3, node(ce.Dst)))
			for k := 0; k+1 < len(route); k += 3 {
				route[k] += pad
				route[k+1] += pad
			}
			attach(route, 0, ce.Src)
			attach(route, len(route)-3, ce.Dst)
			if composite, ok := anchors[i]; ok && len(route) >= 6 {
				g := &cnodes[composite]
				inward := views[ce.Src].kind == kHidden
				at := len(route) - 3
				if inward {
					at = 0
				}
				near := (g.Dir == graphs.TB || g.Dir == graphs.LR) == inward
				var point []float64
				if vertical(g.Dir) {
					py := g.Y + pad + g.H/2
					if near {
						py = g.Y + pad + (views[composite].titleH - g.H/2)
					}
					point = []float64{route[at], py, 0}
				} else {
					px := g.X + pad + g.W/2
					if near {
						px = g.X + pad + -g.W/2
					}
					point = []float64{px, route[at+1], 1}
				}
				if inward {
					route = append(point, route...)
				} else {
					route = append(route, point...)
				}
			}
			path = draw.RoutePath(route, curve, 0, endType.Trim(), leaves, arrives)
		}
		it := scene.Item{
			Kind: scene.Edge, Role: RoleTransition, ID: edge.ID,
			Box:   path.Path.Bounds(),
			Parts: []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:  path.Ends("", endType),
			// Pele draws the edges in a group with round caps.
			Style: style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
		}
		if note {
			it.Role = RoleNoteEdge
			it.Style = style.Style{Stroke: style.Border, Fill: style.None, Dash: style.Dashes(3, 4), Cap: style.CapRound}
		}
		for k := 0; k+1 < len(route); k += 3 {
			it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
		}
		if len(it.Route) == 0 {
			// A self-loop, which the layout gives no route: the corners its curve is drawn about.
			it.Route = it.Parts[0].Path.Corners()
		}
		edges = append(edges, it)
		if label.Width > 0 {
			lx, ly := ce.LabelX+pad, ce.LabelY+pad
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleEdgeLabel, ID: edge.ID,
				Box:   scene.Rect{X: lx - (label.Width+8)/2, Y: ly - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts: []scene.Part{draw.LabelPatch(label, lx, ly)},
				Label: label,
				At:    scene.Point{X: lx, Y: ly},
				Style: style.Style{Fill: style.Bg, Stroke: style.None, Text: style.Fg},
			})
		}
	}

	sc := &scene.Scene{
		Type:     "state",
		Width:    width,
		Height:   height,
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	for _, layer := range [][]scene.Item{clusters, dividers, edges, labels, nodes, notes} {
		sc.Items = append(sc.Items, layer...)
	}
	common.WithTitle(sc, db.Title, RoleTitle, measure, size, pad)
	return sc
}

// titleX picks where a composite's title sits in its title bar so that edges entering from
// above do not cross it, as Pele's titleX does.
func titleX(c *graphs.CNode, width float64, crossings map[float64][]float64) float64 {
	xs, ok := crossings[jsmath.Round(c.Y-c.H/2)]
	if !ok || width == 0 {
		return c.X
	}
	left := c.X - c.W/2
	half := width / 2
	best, bestGap := c.X, -1.0
	for _, x := range [3]float64{c.X, left + groupPad + half, left + c.W - groupPad - half} {
		gap := math.Inf(1)
		for _, cx := range xs {
			if cx > left && cx < left+c.W {
				gap = min(gap, math.Abs(cx-x)-half)
			}
		}
		if gap >= 6 {
			return x
		}
		if gap > bestGap+0.5 {
			best, bestGap = x, gap
		}
	}
	return best
}
