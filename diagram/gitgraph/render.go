package gitgraph

import (
	"math"
	"slices"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of Pele's git graph render.
const (
	dot  = 7.0
	step = 44.0
	lane = 44.0
	// curve is shorter than a step, so a curve between lanes never runs through a commit on
	// a lane it crosses.
	curve    = 36.0
	gap      = 8.0
	labelGap = 20.0
	tailLen  = 20.0
	notch    = 7.0
	tagGap   = 6.0
	// halfRoot is Math.SQRT1_2.
	halfRoot = 0.7071067811865476
)

// shapes are the names of the types of commit, as Pele's classes give them.
var shapes = [...]string{"normal", "reverse", "highlight", "merge", "cherry-pick"}

// view is a commit as it is laid out, Pele's View.
type view struct {
	commit *Commit
	index  int
	lane   int
	m      float64
	label  *text.Label
	tags   []text.Label
}

// flag is a setting of the diagram's configuration that is a boolean, or fallback.
func flag(config diagram.Config, key string, fallback bool) bool {
	git, _ := config["gitGraph"].(diagram.Config)
	if v, ok := git[key].(bool); ok {
		return v
	}
	return fallback
}

func tagWidth(tag *text.Label) float64 { return notch + 4 + tag.Width + 6 }

// Layout lays out a git graph as a scene, as Pele's renderGit draws it: commits in the order
// they were made along the main axis, each on the lane of its branch, joined to their
// parents. A graph that runs left to right turns to run down where the host's width calls
// for it.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if m.Direction != "LR" {
		return lay(m, config, o, m.Direction)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene {
		if down {
			return lay(m, config, o, "TB")
		}
		return lay(m, config, o, "LR")
	})
}

// turned is a commit's label turned about a point, to be placed once the graph is moved to
// where it is on the page.
type turned struct {
	item   int
	angle  float64
	ax, ay float64 // the point it turns about
	cx     float64 // the middle of the label, across, before it turns
	back   scene.Rect
}

// lay is Pele's draw.
func lay(m *Model, config diagram.Config, o *layout.Options, direction string) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.75)
	measure := o.Measure()
	pad := o.Pad()

	showBranches := flag(config, "showBranches", true)
	showCommitLabel := flag(config, "showCommitLabel", true)
	rotate := flag(config, "rotateCommitLabel", true)
	parallel := flag(config, "parallelCommits", false)
	vertical := direction != "LR"
	flip := direction == "BT"

	laneOf := map[string]int{}
	for i, name := range m.Lanes {
		laneOf[name] = i
	}
	laneCount := len(m.Lanes)
	medium := jsmath.Round(size * 0.875)
	names := make([]text.Label, laneCount)
	for i, name := range m.Lanes {
		if !showBranches {
			name = ""
		}
		names[i] = common.PlainLabel(name, measure, medium, 4000, 0)
	}

	commits := slices.Clone(m.Commits)
	slices.SortStableFunc(commits, func(a, b *Commit) int { return a.Seq - b.Seq })
	views := make([]*view, 0, len(commits))
	byID := map[string]*view{}
	for _, c := range commits {
		labelled := showCommitLabel && c.Type != CherryPick && (c.Type != MergeType || c.CustomID != nil && *c.CustomID)
		var tags []text.Label
		for _, tag := range c.Tags {
			if l := common.PlainLabel(tag, measure, small, 4000, 0); len(l.Lines) > 0 {
				tags = append(tags, l)
			}
		}
		v := &view{commit: c, index: len(views), lane: laneOf[c.Branch], tags: tags}
		if labelled {
			l := common.PlainLabel(c.ID, measure, small, 4000, 0)
			v.label = &l
		}
		views = append(views, v)
		byID[c.ID] = v
	}

	// Room each commit needs: along the main axis on either side, and across it towards the
	// previous lane (lo) and the next one (hi). Tags sit on one side of a lane and ids on the
	// other.
	half := make([]float64, len(views))
	tagHalf := make([]float64, len(views))
	lo := make([]float64, laneCount)
	hi := make([]float64, laneCount)
	for i := range lo {
		lo[i], hi[i] = dot, dot
	}
	for k, v := range views {
		tagsMain, tagsCross := 0.0, 0.0
		for t := range v.tags {
			tag := &v.tags[t]
			w := tagWidth(tag)
			if vertical {
				tagsMain = max(tagsMain, tag.Height)
				tagsCross += w + 4
			} else {
				tagsMain = max(tagsMain, w)
				tagsCross += tag.Height + 3
			}
		}
		labelMain, labelCross := 0.0, 0.0
		if l := v.label; l != nil && len(l.Lines) > 0 {
			if rotate {
				labelCross = 3 + float64(halfRoot*(l.Width+l.Height))
			} else {
				if vertical {
					labelMain = l.Height
					labelCross = 6 + l.Width
				} else {
					labelMain = l.Width
					labelCross = 6 + l.Height
				}
			}
		}
		half[k] = max(dot, tagsMain/2, labelMain/2)
		tagHalf[k] = tagsMain / 2
		tagSide := dot
		if len(v.tags) > 0 {
			tagSide = dot + (tagGap + tagsCross)
		}
		labelSide := dot + labelCross
		if vertical {
			lo[v.lane] = max(lo[v.lane], labelSide)
			hi[v.lane] = max(hi[v.lane], tagSide)
		} else {
			lo[v.lane] = max(lo[v.lane], tagSide)
			hi[v.lane] = max(hi[v.lane], labelSide)
		}
	}

	// In a left-to-right graph the tags stand over their commit. A connector that comes down
	// from a lane above joins the commit's lane this far before the commit, clear of the
	// tags, and the commit is placed far enough along to leave room for that.
	lead := make([]float64, len(views))
	if !vertical {
		for k, v := range views {
			if tagHalf[k] == 0 {
				continue
			}
			for _, id := range v.commit.Parents {
				if parent, ok := byID[id]; ok && parent.lane < v.lane {
					lead[k] = tagHalf[k] + 4
				}
			}
		}
	}
	space := func(p, k int) float64 {
		third := 0.0
		if lead[k] > 0 {
			third = half[p] + curve + lead[k]
		}
		return max(step, half[p]+half[k]+gap, third)
	}

	for k, v := range views {
		if !parallel {
			if k > 0 {
				v.m = views[k-1].m + space(k-1, k)
			}
			continue
		}
		// Mermaid's parallelCommits: a commit sits one step after its furthest parent, so
		// branches that grow side by side share positions instead of taking turns.
		mm := 0.0
		for _, id := range v.commit.Parents {
			parent, ok := byID[id]
			if !ok || parent.index >= k {
				continue
			}
			mm = max(mm, parent.m+space(parent.index, k))
		}
		v.m = mm
	}
	end := 0.0
	for _, v := range views {
		end = max(end, v.m)
	}

	cross := make([]float64, laneCount)
	for i := 0; i < laneCount; i++ {
		name := &names[i]
		reach := 0.0
		if len(name.Lines) > 0 {
			if vertical {
				reach = name.Width/2 + 8
			} else {
				reach = name.Height/2 + 2
			}
		}
		lo[i] = max(lo[i], reach)
		hi[i] = max(hi[i], reach)
		if i > 0 {
			cross[i] = cross[i-1] + max(lane, hi[i-1]+lo[i]+gap)
		}
	}

	px := func(m, c float64) float64 {
		if vertical {
			return c
		}
		return m
	}
	py := func(m, c float64) float64 {
		if vertical {
			if flip {
				return -m
			}
			return m
		}
		return c
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	grow := func(x0, y0, x1, y1 float64) {
		if x0 < minX {
			minX = x0
		}
		if y0 < minY {
			minY = y0
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
	}

	var items []scene.Item
	if showBranches {
		for i := 0; i < laneCount; i++ {
			name := &names[i]
			color := style.Series(i)
			c := cross[i]
			w := name.Width + 16
			h := name.Height + 4
			along := w
			if vertical {
				along = h
			}
			from := -labelGap
			if len(name.Lines) > 0 {
				from = -labelGap - along
			}
			cx := px(from+along/2, c)
			cy := py(from+along/2, c)
			grow(px(-labelGap, c), py(-labelGap, c), px(-labelGap, c), py(-labelGap, c))
			grow(px(end+tailLen, c), py(end+tailLen, c), px(end+tailLen, c), py(end+tailLen, c))
			path := new(scene.Path).MoveTo(px(-labelGap, c), py(-labelGap, c)).LineTo(px(end+tailLen, c), py(end+tailLen, c))
			items = append(items, scene.Item{
				Kind: scene.Mark, Role: "gitgraph.lane", ID: m.Lanes[i],
				Box:   path.Bounds(),
				Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: color, Dash: style.Dashes(2, 4)},
			})
			if len(name.Lines) > 0 {
				grow(cx-w/2, cy-h/2, cx+w/2, cy+h/2)
				box := scene.Rect{X: cx - w/2, Y: cy - h/2, W: w, H: h}
				items = append(items, scene.Item{
					Kind: scene.Node, Role: "gitgraph.branch", ID: m.Lanes[i],
					Box:   box,
					Parts: []scene.Part{{Box: box, Rounded: true, Paint: scene.FillOnly}},
					Label: name, At: scene.Point{X: cx, Y: cy},
					Style: style.Style{Fill: color, Stroke: style.None, Text: style.Bg},
				})
			}
		}
	}

	// Positions of the commits on each lane, to tell whether a connector may run along it.
	taken := make([][]float64, laneCount)
	for _, v := range views {
		taken[v.lane] = append(taken[v.lane], v.m)
	}
	if parallel {
		for _, list := range taken {
			slices.Sort(list)
		}
	}
	busy := func(ln int, from, to float64) bool {
		list := taken[ln]
		low, high := 0, len(list)
		for low < high {
			mid := (low + high) >> 1
			if list[mid] <= from {
				low = mid + 1
			} else {
				high = mid
			}
		}
		return low < len(list) && list[low] < to
	}

	var curves, lines []scene.Item
	for _, v := range views {
		c2 := cross[v.lane]
		m2 := v.m
		for j, id := range v.commit.Parents {
			parent, ok := byID[id]
			if !ok || parent == v {
				continue
			}
			c1 := cross[parent.lane]
			m1 := parent.m
			p := &edgePath{px: px, py: py}
			p.move(m1, c1)
			if parent.lane == v.lane {
				p.line(m2, c2)
				lines = append(lines, p.item("", style.Series(v.lane), false))
				continue
			}
			merged := j > 0 && v.commit.Type == MergeType
			picked := j > 0 && !merged
			if m2 <= m1 {
				mid := (m1 + m2) / 2
				p.cube(mid, c1, mid, c2, m2, c2)
			} else if !merged && !busy(v.lane, m1, m2) {
				// Leaves the parent at once and runs along the child's lane.
				s := min(curve, m2-m1)
				p.cube(m1+s/2, c1, m1+s/2, c2, m1+s, c2)
				p.line(m2, c2)
			} else {
				// Turns in just before the child, or before its tags when it comes from
				// their side.
				to := m2
				if parent.lane < v.lane && m2-lead[v.index] > m1 {
					to = m2 - lead[v.index]
				}
				s := min(curve, to-m1)
				along := c1
				if !merged || busy(parent.lane, m1, m2) {
					// Neither lane is free to run along. Between two lanes, past the ids of
					// one and before the tags of the next, a strip stays empty, and the
					// connector takes that.
					if c1 > c2 {
						along = c2 + hi[v.lane] + gap/2
					} else {
						along = c2 - lo[v.lane] - gap/2
					}
					s = min(curve, (to-m1)/2)
					p.cube(m1+s/2, c1, m1+s/2, along, m1+s, along)
				}
				p.line(to-s, along)
				p.cube(to-s/2, along, to-s/2, c2, to, c2)
				if to < m2 {
					p.line(m2, c2)
				}
			}
			shape, color := "", style.Series(v.lane)
			switch {
			case merged:
				shape, color = "merge", style.Series(parent.lane)
			case picked:
				shape = "cherry-pick"
			}
			curves = append(curves, p.item(shape, color, picked))
		}
	}
	items = append(items, curves...)
	items = append(items, lines...)

	var turns []turned
	for _, v := range views {
		c := v.commit
		x := px(v.m, cross[v.lane])
		y := py(v.m, cross[v.lane])
		typ := c.Type
		if c.CustomType != nil {
			typ = *c.CustomType
		}
		grow(x-dot-1, y-dot-1, x+dot+1, y+dot+1)
		items = append(items, symbol(c, typ, x, y, style.Series(v.lane))...)

		if l := v.label; l != nil && len(l.Lines) > 0 {
			w, h := l.Width, l.Height
			back := func(cx, cy float64) scene.Rect {
				return scene.Rect{X: cx - w/2 - 2, Y: cy - h/2 + 2, W: w + 4, H: h - 4}
			}
			it := scene.Item{
				Kind: scene.Text, Role: "gitgraph.commit-label", ID: c.ID, Label: l,
				Style: style.Style{Fill: style.Bg, Stroke: style.None, Text: style.Muted, Radius: style.N(3)},
			}
			if rotate {
				// The text ends beside the commit and trails back the way the history came.
				k := float64(halfRoot*h) / 2
				ax, ay := x+3, y+dot+3+k
				if vertical {
					ax, ay = x-dot-3-k, y
				}
				reach := float64(halfRoot*w) + k
				if flip {
					grow(ax-reach, ay-reach, ax+k, ay+k)
				} else {
					grow(ax-reach, ay-k, ax+k, ay+reach)
				}
				angle := -45.0
				if flip {
					angle = 45
				}
				r := back(ax-w/2, ay)
				turns = append(turns, turned{item: len(items), angle: angle, ax: ax, ay: ay, cx: ax - w/2, back: r})
				it.Box = r
				it.At = scene.Point{X: ax - w/2, Y: ay}
				it.Rotate = angle
			} else {
				cx, cy := x, y+dot+6+h/2
				if vertical {
					cx, cy = x-dot-6-w/2, y
				}
				grow(cx-w/2, cy-h/2, cx+w/2, cy+h/2)
				r := back(cx, cy)
				it.Box = r
				it.Parts = []scene.Part{{Box: r, Rounded: true, Paint: scene.FillOnly}}
				it.At = scene.Point{X: cx, Y: cy}
			}
			items = append(items, it)
		}

		// Beside the lane the tags form a row; above it they stack, the first one furthest
		// out. Pele turns a tag's corners round.
		offset := dot + tagGap
		t, dt := len(v.tags)-1, -1
		if vertical {
			t, dt = 0, 1
		}
		for ; t >= 0 && t < len(v.tags); t += dt {
			tag := &v.tags[t]
			w := tagWidth(tag)
			h := tag.Height
			left, cy := x-w/2, y-offset-h/2
			if vertical {
				left, cy = x+offset, y
			}
			if vertical {
				offset += w + 4
			} else {
				offset += h + 3
			}
			grow(left, cy-h/2, left+w, cy+h/2)
			path := new(scene.Path).MoveTo(left, cy).LineTo(left+notch, cy-h/2).LineTo(left+w, cy-h/2).
				LineTo(left+w, cy+h/2).LineTo(left+notch, cy+h/2).Close()
			items = append(items, scene.Item{
				Kind: scene.Node, Role: "gitgraph.tag", ID: c.ID,
				Box:   scene.Rect{X: left, Y: cy - h/2, W: w, H: h},
				Parts: []scene.Part{{Path: path, Paint: scene.FillAndStroke}},
				Label: tag, At: scene.Point{X: left + notch + 4 + tag.Width/2, Y: cy},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg, Join: style.JoinRound},
			}, scene.Item{
				Kind: scene.Mark, Role: "gitgraph.tag-hole", ID: c.ID,
				Box:   scene.Rect{X: left + notch - 1.5, Y: cy - 1.5, W: 3, H: 3},
				Parts: []scene.Part{{Path: new(scene.Path).Ellipse(left+notch, cy, 1.5, 1.5), Paint: scene.FillOnly}},
				Style: style.Style{Fill: style.Border, Stroke: style.None},
			})
		}
	}

	if minX > maxX {
		grow(0, 0, 0, 0)
	}
	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}
	inner := max(maxX-minX, title.Width)
	width := math.Ceil(inner + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)
	dx := pad - minX + (inner-(maxX-minX))/2
	dy := pad + titleHeight - minY

	for i := range items {
		items[i].Move(dx, dy)
	}
	for _, t := range turns {
		place(&items[t.item], t, dx, dy)
	}
	sc := &scene.Scene{Type: "gitGraph", Width: width, Height: height, FontSize: size, Title: m.AccTitle, Desc: m.AccDescr}
	if len(title.Lines) > 0 {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: "gitgraph.title", Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	sc.Items = append(sc.Items, items...)
	return sc
}

// place turns a commit's label about its point, once the graph has been moved by dx, dy to
// where it is on the page: the label's middle, and the box behind it, which becomes a path.
func place(it *scene.Item, t turned, dx, dy float64) {
	ax, ay := t.ax+dx, t.ay+dy
	s := halfRoot
	if t.angle < 0 {
		s = -halfRoot
	}
	turn := func(x, y float64) (float64, float64) {
		vx, vy := x-ax, y-ay
		return ax + float64(vx*halfRoot) - float64(vy*s), ay + float64(vx*s) + float64(vy*halfRoot)
	}
	it.At.X, it.At.Y = turn(t.cx+dx, ay)
	r := scene.Rect{X: t.back.X + dx, Y: t.back.Y + dy, W: t.back.W, H: t.back.H}
	path := new(scene.Path).RoundRect(r, 3)
	for i := 0; i+1 < len(path.Args); i += 2 {
		path.Args[i], path.Args[i+1] = turn(path.Args[i], path.Args[i+1])
	}
	it.Parts = []scene.Part{{Path: path, Paint: scene.FillOnly}}
	it.Box = path.Bounds()
}

// edgePath builds the path of a connector, by points along and across the lanes.
type edgePath struct {
	px, py func(m, c float64) float64
	path   scene.Path
	route  []scene.Point
}

func (e *edgePath) point(m, c float64) (float64, float64) { return e.px(m, c), e.py(m, c) }

func (e *edgePath) move(m, c float64) {
	x, y := e.point(m, c)
	e.path.MoveTo(x, y)
	e.route = append(e.route, scene.Point{X: x, Y: y})
}

func (e *edgePath) line(m, c float64) {
	x, y := e.point(m, c)
	e.path.LineTo(x, y)
	e.route = append(e.route, scene.Point{X: x, Y: y})
}

func (e *edgePath) cube(m1, c1, m2, c2, m, c float64) {
	x1, y1 := e.point(m1, c1)
	x2, y2 := e.point(m2, c2)
	x, y := e.point(m, c)
	e.path.CubeTo(x1, y1, x2, y2, x, y)
	e.route = append(e.route, scene.Point{X: x, Y: y})
}

// item is the connector as an edge of the scene: its shape is "merge" for the line that
// brings in a merged branch, "cherry-pick" for the one from a picked commit, which is dashed,
// and "" for others. Its ends are round, as Pele's group of edges makes them.
func (e *edgePath) item(shape string, color style.Paint, dashed bool) scene.Item {
	path := e.path
	st := style.Style{Stroke: color, StrokeWidth: style.N(2), Cap: style.CapRound}
	if dashed {
		st.Dash = style.Dashes(4, 3)
	}
	return scene.Item{
		Kind: scene.Edge, Role: "gitgraph.edge", Shape: scene.Shape(shape),
		Box:   path.Bounds(),
		Parts: []scene.Part{{Path: &path, Paint: scene.StrokeOnly}},
		Route: e.route,
		Style: st,
	}
}

// symbol is a commit drawn as its type, Pele's symbol: a dot, a dot crossed out for a
// reverse, a square for a highlight, a ring about a dot for a merge, and a dot with a cherry
// for a cherry-pick. The marks drawn over a dot, in the color of the background, are an item
// of their own: a reverse's cross with round ends and a cherry-pick's stem with a round
// corner, as Pele draws them. A merge's role is gitgraph.merge whatever type it is drawn as.
func symbol(c *Commit, typ int, x, y float64, color style.Paint) []scene.Item {
	role := scene.Role("gitgraph.commit")
	if c.Type == MergeType {
		role = "gitgraph.merge"
	}
	shape := shapes[0]
	if typ >= 0 && typ < len(shapes) {
		shape = shapes[typ]
	}
	tooltip := ""
	// Merges and cherry-picks carry a message Mermaid makes up; only a written one is worth
	// showing.
	if c.Type < MergeType && c.Message != "" {
		tooltip = text.DecodeEntities(c.Message)
	}
	it := scene.Item{
		Kind: scene.Node, Role: role, Shape: scene.Shape(shape), ID: c.ID, Tooltip: tooltip,
		Box:   scene.Rect{X: x - dot, Y: y - dot, W: 2 * dot, H: 2 * dot},
		Style: style.Style{Fill: color, Stroke: style.None},
	}
	dotPart := scene.Part{Path: new(scene.Path).Ellipse(x, y, dot, dot), Paint: scene.FillOnly}
	switch typ {
	case Highlight:
		box := scene.Rect{X: x - dot - 1, Y: y - dot - 1, W: 2*dot + 2, H: 2*dot + 2}
		it.Box = box
		it.Parts = []scene.Part{{Box: box, Rounded: true, Paint: scene.FillOnly}}
		it.Style.Radius = style.N(2)
		return []scene.Item{it}
	case MergeType:
		it.Style.Stroke = color
		it.Style.StrokeWidth = style.N(2)
		it.Parts = []scene.Part{
			{Path: new(scene.Path).Ellipse(x, y, dot-1, dot-1), Paint: scene.Hollow},
			{Path: new(scene.Path).Ellipse(x, y, 2.5, 2.5), Paint: scene.FillOnly},
		}
		return []scene.Item{it}
	case Reverse:
		it.Parts = []scene.Part{dotPart}
		cross := new(scene.Path).MoveTo(x-3, y-3).LineTo(x-3+6, y-3+6).MoveTo(x-3, y+3).LineTo(x-3+6, y+3-6)
		return []scene.Item{it, {
			Kind: scene.Mark, Role: "gitgraph.commit-mark", Shape: "reverse", ID: c.ID,
			Box:   cross.Bounds(),
			Parts: []scene.Part{{Path: cross, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Bg, StrokeWidth: style.N(1.5), Cap: style.CapRound},
		}}
	case CherryPick:
		it.Parts = []scene.Part{dotPart}
		stem := new(scene.Path).MoveTo(x-2.6, y+1).LineTo(x+0.8, y-4.2).LineTo(x+2.6, y+1)
		return []scene.Item{it, {
			Kind: scene.Mark, Role: "gitgraph.commit-mark", Shape: "cherry-pick", ID: c.ID,
			Box: scene.Rect{X: x - 2.6 - 1.8, Y: y - 4.2, W: 2*2.6 + 2*1.8, H: 4.2 + 2.2 + 1.8},
			Parts: []scene.Part{
				{Path: stem, Paint: scene.StrokeOnly},
				{Path: new(scene.Path).Ellipse(x-2.6, y+2.2, 1.8, 1.8), Paint: scene.FillOnly},
				{Path: new(scene.Path).Ellipse(x+2.6, y+2.2, 1.8, 1.8), Paint: scene.FillOnly},
			},
			Style: style.Style{Fill: style.Bg, Stroke: style.Bg, Join: style.JoinRound},
		}}
	}
	it.Parts = []scene.Part{dotPart}
	return []scene.Item{it}
}
