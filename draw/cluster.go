package draw

import "math"

// Crossings are the x of every route point, by its y rounded: the edges that cross a
// cluster's top border. It is Pele's Crossings.
type Crossings map[float64][]float64

// MarkCrossings adds the points of a route of x, y, axis triples, as Pele's markCrossings
// does.
func MarkCrossings(c Crossings, route []float64) {
	for k := 0; k+1 < len(route); k += 3 {
		key := jsRound(route[k+1])
		c[key] = append(c[key], route[k])
	}
}

// ClusterTitleX is where along a cluster's top a title width wide is centered, so that the
// edges that enter there do not cross it, as Pele's clusterTitleX picks it: at the left,
// in the middle, or at the right, pad in from the sides less 6, the first that is 6 clear
// of every edge, or else the clearest.
func ClusterTitleX(c Node, width float64, crossings Crossings, pad float64) float64 {
	if width == 0 {
		return c.X
	}
	top := c.Y - c.H/2
	left := c.X - c.W/2
	half := width / 2
	candidates := [3]float64{left + pad - 6 + half, c.X, left + c.W - pad + 6 - half}
	xs, ok := crossings[jsRound(top)]
	if !ok {
		return candidates[0]
	}
	best, bestGap := candidates[0], -1.0
	for _, x := range candidates {
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

// TitleCrossed reports whether an edge that enters a cluster at its top runs through its
// title, width wide and centered at x, as Pele's titleCrossed does.
func TitleCrossed(c Node, width, x float64, crossings Crossings) bool {
	if !(width > 0) {
		return false
	}
	xs, ok := crossings[jsRound(c.Y-c.H/2)]
	if !ok {
		return false
	}
	left := c.X - c.W/2
	for _, cx := range xs {
		if cx > left && cx < left+c.W && math.Abs(cx-x) < width/2+4 {
			return true
		}
	}
	return false
}

// A title that an edge runs through is drawn over the edges, as Pele's struckTitle draws
// it: the text on an outline StruckHalo wide, with round joins, in the color of the cluster
// behind it, so that only the line is seen to stop short of the letters. Each cluster is
// half its fill over what is behind it, so the color is the theme's accent, StruckTint
// percent of it over the background.
const StruckHalo = 5

// StruckDepth is how many clusters deep the cluster at index is, itself counted and no more
// than 8, as Pele's struckTitle counts it; parent gives a cluster's parent, or a negative
// number for none.
func StruckDepth(parent func(i int) int, index int) int {
	depth := 1
	for p := parent(index); p >= 0 && depth < 8; p = parent(p) {
		depth++
	}
	return depth
}

// StruckTint is the percentage of the theme's accent in the outline of a struck title
// depth clusters deep.
func StruckTint(depth int) float64 {
	return 100 * (1 - math.Pow(0.5, float64(depth)))
}
