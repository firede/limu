package graphs

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/jsmath"
)

// The fixture is what Pele makes of the same graphs; ./scripts/oracle.sh writes it.

// num is a number of the fixture, where the infinities and NaN are written as strings.
type num float64

func (n *num) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		switch s {
		case "Infinity":
			*n = num(math.Inf(1))
		case "-Infinity":
			*n = num(math.Inf(-1))
		case "NaN":
			*n = num(math.NaN())
		default:
			return fmt.Errorf("number %q", s)
		}
		return nil
	}
	f, err := strconv.ParseFloat(string(data), 64)
	*n = num(f)
	return err
}

type peleOptions struct {
	NodeSep  num  `json:"nodeSep"`
	EdgeSep  num  `json:"edgeSep"`
	RankSep  num  `json:"rankSep"`
	PortSep  num  `json:"portSep"`
	Tracks   bool `json:"tracks"`
	Straight bool `json:"straight"`
	Apart    bool `json:"apart"`
	HeadRoom num  `json:"headRoom"`
}

func (o peleOptions) options() LayeredOptions {
	return LayeredOptions{
		NodeSep: float64(o.NodeSep), EdgeSep: float64(o.EdgeSep), RankSep: float64(o.RankSep), PortSep: float64(o.PortSep),
		Tracks: o.Tracks, Straight: o.Straight, Apart: o.Apart, HeadRoom: float64(o.HeadRoom),
	}
}

type peleResult struct {
	Width  num `json:"width"`
	Height num `json:"height"`
}

type peleLayered struct {
	Name  string `json:"name"`
	Nodes []struct {
		W    num  `json:"w"`
		H    num  `json:"h"`
		Kind kind `json:"kind"`
		Pin  num  `json:"pin"`
		Span num  `json:"span"`
	} `json:"nodes"`
	Edges []struct {
		Tail      int  `json:"tail"`
		Head      int  `json:"head"`
		Minlen    num  `json:"minlen"`
		LabelW    num  `json:"labelW"`
		LabelH    num  `json:"labelH"`
		TailDx    num  `json:"tailDx"`
		HeadDx    num  `json:"headDx"`
		TailFixed bool `json:"tailFixed"`
		HeadFixed bool `json:"headFixed"`
	} `json:"edges"`
	Opt peleOptions `json:"opt"`
	Out struct {
		Result peleResult `json:"result"`
		Nodes  []struct {
			X    num `json:"x"`
			Y    num `json:"y"`
			Rank int `json:"rank"`
		} `json:"nodes"`
		Edges []struct {
			TailDx   num   `json:"tailDx"`
			HeadDx   num   `json:"headDx"`
			Reversed bool  `json:"reversed"`
			Points   []num `json:"points"`
			LabelX   num   `json:"labelX"`
			LabelY   num   `json:"labelY"`
		} `json:"edges"`
	} `json:"out"`
}

type peleCompound struct {
	Name    string `json:"name"`
	RootDir Dir    `json:"rootDir"`
	Nodes   []struct {
		W         num  `json:"w"`
		H         num  `json:"h"`
		Parent    int  `json:"parent"`
		IsGroup   bool `json:"isGroup"`
		Dir       Dir  `json:"dir"`
		PadX      num  `json:"padX"`
		PadTop    num  `json:"padTop"`
		PadBottom num  `json:"padBottom"`
		MinW      num  `json:"minW"`
		Seq       num  `json:"seq"`
		Span      num  `json:"span"`
	} `json:"nodes"`
	Edges []struct {
		Src    int `json:"src"`
		Dst    int `json:"dst"`
		Minlen num `json:"minlen"`
		LabelW num `json:"labelW"`
		LabelH num `json:"labelH"`
	} `json:"edges"`
	Opt   peleOptions `json:"opt"`
	Shift []num       `json:"shift"`
	Out   struct {
		Result peleResult `json:"result"`
		Nodes  []struct {
			X      num `json:"x"`
			Y      num `json:"y"`
			W      num `json:"w"`
			H      num `json:"h"`
			Parent int `json:"parent"`
			Seq    num `json:"seq"`
		} `json:"nodes"`
		Edges []struct {
			Src    int   `json:"src"`
			Dst    int   `json:"dst"`
			Route  []num `json:"route"`
			LabelX num   `json:"labelX"`
			LabelY num   `json:"labelY"`
		} `json:"edges"`
	} `json:"out"`
}

type peleLayout struct {
	Ease []struct {
		Across num `json:"across"`
		Along  num `json:"along"`
		Ease   num `json:"ease"`
	} `json:"ease"`
	Hypot []struct {
		A     num `json:"a"`
		B     num `json:"b"`
		Hypot num `json:"hypot"`
	} `json:"hypot"`
	Layered  []peleLayered  `json:"layered"`
	Compound []peleCompound `json:"compound"`
}

var pele *peleLayout

func loadPele(t testing.TB) *peleLayout {
	t.Helper()
	if pele != nil {
		return pele
	}
	var p peleLayout
	if err := fixture.Load("testdata/pele/layout.json", &p); err != nil {
		t.Fatal(err)
	}
	pele = &p
	return pele
}

// same reports whether two numbers are the same, NaN being the same as NaN.
func same(got float64, want num) bool {
	return got == float64(want) || (got != got && want != want)
}

// diff collects the first few differences of one case.
type diff struct {
	lines []string
}

func (d *diff) num(what string, got float64, want num) {
	if !same(got, want) {
		d.add("%s = %v, Pele %v", what, got, float64(want))
	}
}

func (d *diff) nums(what string, got []float64, want []num) {
	if len(got) != len(want) {
		d.add("%s has %d numbers, Pele %d\n   got %v\n  Pele %v", what, len(got), len(want), got, want)
		return
	}
	for i := range got {
		if !same(got[i], want[i]) {
			d.add("%s[%d] = %v, Pele %v\n   got %v\n  Pele %v", what, i, got[i], float64(want[i]), got, want)
			return
		}
	}
}

func (d *diff) add(format string, args ...any) {
	if len(d.lines) < 6 {
		d.lines = append(d.lines, fmt.Sprintf(format, args...))
	}
}

func (d *diff) report(t *testing.T, name string) {
	t.Helper()
	if len(d.lines) == 0 {
		return
	}
	msg := name + ":"
	for _, l := range d.lines {
		msg += "\n  " + l
	}
	t.Error(msg)
}

func TestEaseMatchesPele(t *testing.T) {
	for _, c := range loadPele(t).Ease {
		if got := Ease(float64(c.Across), float64(c.Along)); !same(got, c.Ease) {
			t.Errorf("Ease(%v, %v) = %v, Pele %v", c.Across, c.Along, got, c.Ease)
		}
	}
}

func TestHypotMatchesV8(t *testing.T) {
	for _, c := range loadPele(t).Hypot {
		if got := jsmath.Hypot(float64(c.A), float64(c.B)); !same(got, c.Hypot) {
			t.Errorf("jsmath.Hypot(%v, %v) = %v, Math.hypot %v", c.A, c.B, got, c.Hypot)
		}
	}
}

func layeredInput(c *peleLayered) ([]lNode, []lEdge) {
	nodes := make([]lNode, len(c.Nodes))
	for i, n := range c.Nodes {
		nodes[i] = newLNode(float64(n.W), float64(n.H), n.Kind)
		nodes[i].Pin = float64(n.Pin)
		nodes[i].Span = float64(n.Span)
	}
	edges := make([]lEdge, len(c.Edges))
	for i, e := range c.Edges {
		edges[i] = newLEdge(e.Tail, e.Head, float64(e.Minlen), float64(e.LabelW), float64(e.LabelH))
		edges[i].TailDx = float64(e.TailDx)
		edges[i].HeadDx = float64(e.HeadDx)
		edges[i].TailFixed = e.TailFixed
		edges[i].HeadFixed = e.HeadFixed
	}
	return nodes, edges
}

func TestLayeredMatchesPele(t *testing.T) {
	cases := loadPele(t).Layered
	if len(cases) < 300 {
		t.Fatalf("only %d cases", len(cases))
	}
	for i := range cases {
		c := &cases[i]
		nodes, edges := layeredInput(c)
		got := layered(nodes, edges, c.Opt.options())
		var d diff
		d.num("width", got.Width, c.Out.Result.Width)
		d.num("height", got.Height, c.Out.Result.Height)
		for i, n := range nodes {
			w := c.Out.Nodes[i]
			d.num(fmt.Sprintf("node %d x", i), n.X, w.X)
			d.num(fmt.Sprintf("node %d y", i), n.Y, w.Y)
			if n.Rank != w.Rank {
				d.add("node %d rank = %d, Pele %d", i, n.Rank, w.Rank)
			}
		}
		for i, e := range edges {
			w := c.Out.Edges[i]
			if e.Reversed != w.Reversed {
				d.add("edge %d reversed = %v, Pele %v", i, e.Reversed, w.Reversed)
			}
			d.num(fmt.Sprintf("edge %d tailDx", i), e.TailDx, w.TailDx)
			d.num(fmt.Sprintf("edge %d headDx", i), e.HeadDx, w.HeadDx)
			d.nums(fmt.Sprintf("edge %d points", i), e.Points, w.Points)
			d.num(fmt.Sprintf("edge %d labelX", i), e.LabelX, w.LabelX)
			d.num(fmt.Sprintf("edge %d labelY", i), e.LabelY, w.LabelY)
		}
		d.report(t, c.Name)
	}
}

func compoundInput(c *peleCompound) ([]CNode, []CEdge) {
	nodes := make([]CNode, len(c.Nodes))
	for i, n := range c.Nodes {
		nodes[i] = NewCNode(float64(n.W), float64(n.H), n.Parent)
		nodes[i].IsGroup = n.IsGroup
		nodes[i].Dir = n.Dir
		nodes[i].PadX = float64(n.PadX)
		nodes[i].PadTop = float64(n.PadTop)
		nodes[i].PadBottom = float64(n.PadBottom)
		nodes[i].MinW = float64(n.MinW)
		nodes[i].Seq = float64(n.Seq)
		nodes[i].Span = float64(n.Span)
	}
	edges := make([]CEdge, len(c.Edges))
	for i, e := range c.Edges {
		edges[i] = CEdge{Src: e.Src, Dst: e.Dst, Minlen: float64(e.Minlen), LabelW: float64(e.LabelW), LabelH: float64(e.LabelH)}
	}
	return nodes, edges
}

func TestCompoundMatchesPele(t *testing.T) {
	cases := loadPele(t).Compound
	if len(cases) < 300 {
		t.Fatalf("only %d cases", len(cases))
	}
	for i := range cases {
		c := &cases[i]
		nodes, edges := compoundInput(c)
		got := Compound(nodes, edges, c.RootDir, c.Opt.options())
		if c.Shift != nil {
			Shift(nodes, edges, float64(c.Shift[0]), float64(c.Shift[1]))
		}
		var d diff
		d.num("width", got.Width, c.Out.Result.Width)
		d.num("height", got.Height, c.Out.Result.Height)
		for i, n := range nodes {
			w := c.Out.Nodes[i]
			d.num(fmt.Sprintf("node %d x", i), n.X, w.X)
			d.num(fmt.Sprintf("node %d y", i), n.Y, w.Y)
			d.num(fmt.Sprintf("node %d w", i), n.W, w.W)
			d.num(fmt.Sprintf("node %d h", i), n.H, w.H)
			d.num(fmt.Sprintf("node %d seq", i), n.Seq, w.Seq)
			if n.Parent != w.Parent {
				d.add("node %d parent = %d, Pele %d", i, n.Parent, w.Parent)
			}
		}
		for i, e := range edges {
			w := c.Out.Edges[i]
			if e.Src != w.Src || e.Dst != w.Dst {
				d.add("edge %d runs %d to %d, Pele %d to %d", i, e.Src, e.Dst, w.Src, w.Dst)
			}
			d.nums(fmt.Sprintf("edge %d route", i), e.Route, w.Route)
			d.num(fmt.Sprintf("edge %d labelX", i), e.LabelX, w.LabelX)
			d.num(fmt.Sprintf("edge %d labelY", i), e.LabelY, w.LabelY)
		}
		d.report(t, c.Name)
	}
}

// BenchmarkLayered lays out a random graph of 500 nodes and about 750 edges, a fifth of them
// labelled, with edges spread along their nodes.
func BenchmarkLayered(b *testing.B) {
	nodes, edges := benchGraph(500)
	opt := LayeredOptions{NodeSep: 40, EdgeSep: 16, RankSep: 48, PortSep: 20, HeadRoom: 7}
	ns := make([]lNode, len(nodes))
	es := make([]lEdge, len(edges))
	b.ReportAllocs()
	for b.Loop() {
		copy(ns, nodes)
		copy(es, edges)
		layered(ns, es, opt)
	}
}

// BenchmarkCompound lays out a random graph of 500 nodes, some of them in groups.
func BenchmarkCompound(b *testing.B) {
	ln, le := benchGraph(500)
	nodes := make([]CNode, len(ln))
	for i, n := range ln {
		nodes[i] = NewCNode(n.W, n.H, -1)
		nodes[i].Span = n.Span
		nodes[i].Seq = float64(i)
		if i%25 == 0 {
			nodes[i] = NewCNode(0, 0, -1)
			nodes[i].IsGroup = true
			nodes[i].PadX, nodes[i].PadTop, nodes[i].PadBottom, nodes[i].MinW = 20, 20, 20, 28
			nodes[i].Span = math.Inf(1)
			nodes[i].Seq = float64(i)
		}
	}
	for i := range nodes {
		if g := i / 25 * 25; g != i && i%3 != 0 {
			nodes[i].Parent = g
		}
	}
	edges := make([]CEdge, len(le))
	for i, e := range le {
		edges[i] = CEdge{Src: e.Tail, Dst: e.Head, Minlen: 1, LabelW: e.LabelW, LabelH: e.LabelH}
	}
	opt := LayeredOptions{NodeSep: 40, EdgeSep: 16, RankSep: 48, PortSep: 20, HeadRoom: 7}
	ns := make([]CNode, len(nodes))
	es := make([]CEdge, len(edges))
	b.ReportAllocs()
	for b.Loop() {
		copy(ns, nodes)
		copy(es, edges)
		Compound(ns, es, TB, opt)
	}
}

func benchGraph(count int) ([]lNode, []lEdge) {
	s := uint32(500)
	next := func() float64 {
		s = s*1664525 + 1013904223
		return float64(s) / (1 << 32)
	}
	nodes := make([]lNode, count)
	for i := range nodes {
		w := 40 + next()*160
		nodes[i] = newLNode(w, 30+next()*40, kindNode)
		nodes[i].Span = w - 10
	}
	var edges []lEdge
	for k := 0; k < count*3/2; k++ {
		tail := int(next() * float64(count))
		head := int(next() * float64(count-1))
		if head >= tail {
			head++
		}
		e := newLEdge(tail, head, 1, 0, 0)
		if next() < 0.2 {
			e.LabelW, e.LabelH = 20+next()*80, 23
		}
		edges = append(edges, e)
	}
	return nodes, edges
}
