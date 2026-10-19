package layout

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele's laneLayout makes of the same graphs; ./scripts/oracle.sh writes
// it with the lanes suite.

type peleLanes struct {
	Cases []struct {
		Name  string `json:"name"`
		Dir   Dir    `json:"dir"`
		Nodes []struct {
			W         num  `json:"w"`
			H         num  `json:"h"`
			Parent    int  `json:"parent"`
			IsGroup   bool `json:"isGroup"`
			PadX      num  `json:"padX"`
			PadTop    num  `json:"padTop"`
			PadBottom num  `json:"padBottom"`
			MinW      num  `json:"minW"`
		} `json:"nodes"`
		Edges []struct {
			Src    int `json:"src"`
			Dst    int `json:"dst"`
			Minlen num `json:"minlen"`
			LabelW num `json:"labelW"`
			LabelH num `json:"labelH"`
		} `json:"edges"`
		LanePad num         `json:"lanePad"`
		Opt     peleOptions `json:"opt"`
		Out     struct {
			Result peleResult `json:"result"`
			Nodes  []struct {
				X      num `json:"x"`
				Y      num `json:"y"`
				W      num `json:"w"`
				H      num `json:"h"`
				PadTop num `json:"padTop"`
			} `json:"nodes"`
			Edges []struct {
				Route  []num `json:"route"`
				LabelX num   `json:"labelX"`
				LabelY num   `json:"labelY"`
			} `json:"edges"`
		} `json:"out"`
	} `json:"cases"`
}

func TestLanesMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/lanes.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleLanes
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 600 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	for _, c := range p.Cases {
		nodes := make([]CNode, len(c.Nodes))
		for i, n := range c.Nodes {
			nodes[i] = NewCNode(float64(n.W), float64(n.H), n.Parent)
			nodes[i].IsGroup = n.IsGroup
			nodes[i].PadX = float64(n.PadX)
			nodes[i].PadTop = float64(n.PadTop)
			nodes[i].PadBottom = float64(n.PadBottom)
			nodes[i].MinW = float64(n.MinW)
		}
		edges := make([]CEdge, len(c.Edges))
		for i, e := range c.Edges {
			edges[i] = CEdge{Src: e.Src, Dst: e.Dst, Minlen: float64(e.Minlen), LabelW: float64(e.LabelW), LabelH: float64(e.LabelH)}
		}
		got := Lanes(nodes, edges, c.Dir, c.Opt.options(), float64(c.LanePad))
		var d diff
		d.num("width", got.Width, c.Out.Result.Width)
		d.num("height", got.Height, c.Out.Result.Height)
		for i, n := range nodes {
			w := c.Out.Nodes[i]
			d.num(fmt.Sprintf("node %d x", i), n.X, w.X)
			d.num(fmt.Sprintf("node %d y", i), n.Y, w.Y)
			d.num(fmt.Sprintf("node %d w", i), n.W, w.W)
			d.num(fmt.Sprintf("node %d h", i), n.H, w.H)
			d.num(fmt.Sprintf("node %d padTop", i), n.PadTop, w.PadTop)
		}
		for i, e := range edges {
			w := c.Out.Edges[i]
			d.nums(fmt.Sprintf("edge %d route", i), e.Route, w.Route)
			d.num(fmt.Sprintf("edge %d labelX", i), e.LabelX, w.LabelX)
			d.num(fmt.Sprintf("edge %d labelY", i), e.LabelY, w.LabelY)
		}
		d.report(t, c.Name)
	}
}
