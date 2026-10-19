package sankey

import (
	"math"
	"slices"

	"github.com/firede/limu/diagram"
)

// alignment is the column a node goes in, Pele's Alignment: "justify", "left", "right" or
// "center".
type alignment string

// layoutOptions are what Pele's layoutSankey is given, its LayoutOptions.
type layoutOptions struct {
	width, height float64
	nodeWidth     float64
	nodePadding   float64
	// labelHeight is the height of a line of a label; thin nodes are kept at least this far
	// apart where there is room.
	labelHeight float64
	alignment   alignment
}

// placedNode is a node laid out, Pele's PlacedNode. value is the total flow through it, in
// the diagram's units.
type placedNode struct {
	layer        int
	value        float64
	x, y, height float64
}

// placedLink is a link laid out, Pele's PlacedLink: sourceY and targetY are the middles of
// its band where it leaves its source and where it enters its target.
type placedLink struct {
	source, target   int
	value, thickness float64
	sourceY, targetY float64
}

// passes is how many times the columns settle, Pele's PASSES.
const passes = 6

// pulls are how far a node moves towards where its bands would run level in each pass:
// Math.pow(0.99, pass) as V8 gives it, for Pele's six passes. Go's math.Pow gives other last
// bits from the third power on.
var pulls = [passes]float64{1, 0.99, 0.9801, 0.970299, 0.96059601, 0.9509900498999999}

// sortBy sorts a list as Pele's sort does: by insertion where it is short, and with the
// built-in sort otherwise. The comparisons Pele gives it order every two items one way, ties
// broken by their index, so any sort puts the list in the same order as V8's.
func sortBy(list []int, compare func(a, b int) float64) {
	if len(list) > 8 {
		slices.SortFunc(list, func(a, b int) int {
			switch c := compare(a, b); {
			case c < 0:
				return -1
			case c > 0:
				return 1
			}
			return 0
		})
		return
	}
	for i := 1; i < len(list); i++ {
		item := list[i]
		j := i - 1
		for j >= 0 && compare(list[j], item) > 0 {
			list[j+1] = list[j]
			j--
		}
		list[j+1] = item
	}
}

// or is JavaScript's a || b of two numbers: b where a is 0 or NaN.
func or(a, b float64) float64 {
	if a == 0 || a != a {
		return b
	}
	return a
}

// layoutSankey places nodes in columns by their distance from a source, sizes them by their
// flow, and then lets each column settle next to its neighbours so that the bands cross
// less, as Pele's layoutSankey does. A diagram whose links run in a circle is an error.
func layoutSankey(m *Model, options layoutOptions) ([]placedNode, []placedLink, error) {
	n := len(m.Nodes)
	links := make([]placedLink, len(m.Links))
	for k, link := range m.Links {
		value := 0.0
		if !math.IsNaN(link.Value) && !math.IsInf(link.Value, 0) && link.Value > 0 {
			value = link.Value
		}
		links[k] = placedLink{source: m.Nodes[link.Source].Index, target: m.Nodes[link.Target].Index, value: value}
	}
	outgoing := make([][]int, n)
	incoming := make([][]int, n)
	for k, link := range links {
		outgoing[link.source] = append(outgoing[link.source], k)
		incoming[link.target] = append(incoming[link.target], k)
	}

	// Longest distance from a source, in topological order; what is left over lies on a
	// cycle.
	depth := make([]int, n)
	pending := make([]int, n)
	var order []int
	for i := 0; i < n; i++ {
		pending[i] = len(incoming[i])
		if pending[i] == 0 {
			order = append(order, i)
		}
	}
	for q := 0; q < len(order); q++ {
		i := order[q]
		for _, k := range outgoing[i] {
			t := links[k].target
			if depth[t] < depth[i]+1 {
				depth[t] = depth[i] + 1
			}
			pending[t]--
			if pending[t] == 0 {
				order = append(order, t)
			}
		}
	}
	if len(order) < n {
		// Every node left over has a predecessor that is left over too; walking back finds
		// the loop.
		stuck := 0
		for pending[stuck] == 0 {
			stuck++
		}
		seen := make([]bool, n)
		for !seen[stuck] {
			seen[stuck] = true
			for _, k := range incoming[stuck] {
				if pending[links[k].source] > 0 {
					stuck = links[k].source
					break
				}
			}
		}
		return nil, nil, diagram.NewError(`Sankey diagrams cannot have circular links, and "`+m.Nodes[stuck].ID+`" is part of one.`,
			diagram.Semantic, "sankey")
	}
	layers := 1
	for i := 0; i < n; i++ {
		if depth[i]+1 > layers {
			layers = depth[i] + 1
		}
	}
	reach := make([]int, n)
	for q := n - 1; q >= 0; q-- {
		i := order[q]
		for _, k := range outgoing[i] {
			if reach[i] < reach[links[k].target]+1 {
				reach[i] = reach[links[k].target] + 1
			}
		}
	}

	// Flows are scaled to the largest one, so that sums of huge values stay finite.
	largest := 0.0
	for _, link := range links {
		if link.value > largest {
			largest = link.value
		}
	}
	unit := 0.0
	if largest > 0 {
		unit = 1 / largest
	}

	nodes := make([]placedNode, 0, n)
	flow := make([]float64, n)
	columns := make([][]int, layers)
	step := 0.0
	if layers > 1 {
		step = (options.width - options.nodeWidth) / float64(layers-1)
	}
	for i := 0; i < n; i++ {
		var layer int
		switch options.alignment {
		case "left":
			layer = depth[i]
		case "right":
			layer = layers - 1 - reach[i]
		case "center":
			layer = depth[i]
			if len(incoming[i]) == 0 && len(outgoing[i]) > 0 {
				layer = layers
				for _, k := range outgoing[i] {
					if depth[links[k].target]-1 < layer {
						layer = depth[links[k].target] - 1
					}
				}
			}
		default:
			layer = layers - 1
			if len(outgoing[i]) > 0 {
				layer = depth[i]
			}
		}
		into, out, value, valueOut := 0.0, 0.0, 0.0, 0.0
		for _, k := range incoming[i] {
			into += float64(links[k].value * unit)
			value += links[k].value
		}
		for _, k := range outgoing[i] {
			out += float64(links[k].value * unit)
			valueOut += links[k].value
		}
		flow[i] = max(into, out)
		nodes = append(nodes, placedNode{layer: layer, value: max(value, valueOut), x: float64(layer) * step})
		columns[layer] = append(columns[layer], i)
	}

	tallest := 1
	for _, column := range columns {
		if len(column) > tallest {
			tallest = len(column)
		}
	}
	gap := options.nodePadding
	if tallest > 1 {
		gap = min(options.nodePadding, options.height/float64(tallest-1))
	}
	scale := math.Inf(1)
	for _, column := range columns {
		sum := 0.0
		for _, i := range column {
			sum += flow[i]
		}
		if sum > 0 {
			scale = min(scale, (options.height-float64(float64(len(column)-1)*gap))/sum)
		}
	}
	if math.IsInf(scale, 0) || math.IsNaN(scale) || scale < 0 {
		scale = 0
	}

	// A thin node still needs room for its label. Where every column can afford it, each node
	// claims at least a label's height, and the flows are scaled down to make that fit.
	claim := max(0, options.labelHeight-gap)
	fits := func(s float64) bool {
		for _, column := range columns {
			extent := float64(len(column)-1) * gap
			for _, i := range column {
				extent += max(flow[i]*s, claim)
			}
			if extent > options.height+1e-6 {
				return false
			}
		}
		return true
	}
	if claim > 0 && !fits(scale) {
		if fits(0) {
			low := 0.0
			for k := 0; k < 40; k++ {
				mid := (low + scale) / 2
				if fits(mid) {
					low = mid
				} else {
					scale = mid
				}
			}
			scale = low
		} else {
			claim = 0
		}
	}

	// The room a node keeps clear above and below itself.
	margin := make([]float64, n)
	for k := range links {
		links[k].thickness = links[k].value * unit * scale
	}
	for _, column := range columns {
		y := 0.0
		for _, i := range column {
			nodes[i].height = float64(flow[i] * scale)
			margin[i] = max(0, claim-nodes[i].height) / 2
			nodes[i].y = y + margin[i]
			y += nodes[i].height + float64(2*margin[i]) + gap
		}
		// Spare room is shared out between the nodes of a column.
		spare := (options.height - y + gap) / float64(len(column)+1)
		for k, i := range column {
			nodes[i].y += float64(spare * float64(k+1))
		}
	}

	center := func(i int) float64 { return nodes[i].y + nodes[i].height/2 }

	// Bands leave and enter a node in the order of where they go, so they do not cross at
	// it. These are the offsets of each band's middle from the top of the node at either end.
	sourceOffset := make([]float64, len(links))
	targetOffset := make([]float64, len(links))
	byTarget := func(a, b int) float64 {
		return or(center(links[a].target)-center(links[b].target), float64(a-b))
	}
	bySource := func(a, b int) float64 {
		return or(center(links[a].source)-center(links[b].source), float64(a-b))
	}
	byCenter := func(a, b int) float64 { return or(center(a)-center(b), float64(a-b)) }
	stackOut := func(i int) {
		y := 0.0
		sortBy(outgoing[i], byTarget)
		for _, k := range outgoing[i] {
			sourceOffset[k] = y + links[k].thickness/2
			y += links[k].thickness
		}
	}
	stackIn := func(i int) {
		y := 0.0
		sortBy(incoming[i], bySource)
		for _, k := range incoming[i] {
			targetOffset[k] = y + links[k].thickness/2
			y += links[k].thickness
		}
	}
	stackAll := func() {
		for i := 0; i < n; i++ {
			stackOut(i)
			stackIn(i)
		}
	}

	// Moves each node of a column towards where its bands would run level, then restores the
	// order and the gaps. forward looks at the bands coming in, otherwise at those going out.
	settle := func(column []int, pull float64, forward bool) {
		for _, i := range column {
			list := outgoing[i]
			if forward {
				list = incoming[i]
			}
			if len(list) == 0 {
				continue
			}
			if forward {
				stackIn(i)
			} else {
				stackOut(i)
			}
			weighted, weight, plain := 0.0, 0.0, 0.0
			for _, k := range list {
				var level float64
				other := links[k].target
				if forward {
					level = nodes[links[k].source].y + sourceOffset[k] - targetOffset[k]
					other = links[k].source
				} else {
					level = nodes[links[k].target].y + targetOffset[k] - sourceOffset[k]
				}
				weighted += float64(level * links[k].thickness)
				weight += links[k].thickness
				plain += center(other) - nodes[i].height/2
			}
			wanted := plain / float64(len(list))
			if weight > 0 {
				wanted = weighted / weight
			}
			nodes[i].y += float64((wanted - nodes[i].y) * pull)
		}
		sortBy(column, byCenter)
		y := 0.0
		for _, i := range column {
			if nodes[i].y-margin[i] < y {
				nodes[i].y = y + margin[i]
			}
			y = nodes[i].y + nodes[i].height + margin[i] + gap
		}
		y = options.height
		for k := len(column) - 1; k >= 0; k-- {
			i := column[k]
			if nodes[i].y+nodes[i].height+margin[i] > y {
				nodes[i].y = y - nodes[i].height - margin[i]
			}
			y = nodes[i].y - margin[i] - gap
		}
	}

	stackAll()
	for pass := 0; pass < passes; pass++ {
		pull := pulls[pass]
		for l := layers - 2; l >= 0; l-- {
			settle(columns[l], pull, false)
		}
		stackAll()
		for l := 1; l < layers; l++ {
			settle(columns[l], pull, true)
		}
		stackAll()
	}
	for k := range links {
		links[k].sourceY = nodes[links[k].source].y + sourceOffset[k]
		links[k].targetY = nodes[links[k].target].y + targetOffset[k]
	}
	return nodes, links, nil
}
