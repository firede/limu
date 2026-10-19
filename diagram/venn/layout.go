package venn

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/scene"
)

// circle is a set's circle, Pele's Circle.
type circle struct {
	id      string
	x, y, r float64
}

// spot is where the content of a region goes, Pele's Spot: the point inside all of its
// circles and outside every other that is farthest from their outlines, and how far that is.
// A margin of 0 or less means the circles as placed leave the region empty.
type spot struct {
	x, y, margin float64
}

const tau = math.Pi * 2

// setsKey is one key for a list of set ids, for the sets of a subset as Pele's
// JSON.stringify writes them.
func setsKey(ids []string) string {
	var b strings.Builder
	for _, id := range ids {
		b.WriteString(strconv.Itoa(len(id)))
		b.WriteByte(':')
		b.WriteString(id)
	}
	return b.String()
}

// ensurePairwiseSubsets adds, for every overlap of three or more sets, the pairwise overlaps
// it implies and the diagram does not state, a quarter of the smaller set each, so that the
// circles are drawn overlapping, as Pele's ensurePairwiseSubsets does. It returns the list it
// was given when there is nothing to add.
func ensurePairwiseSubsets(subsets []Subset) []Subset {
	present := map[string]bool{}
	sizes := map[string]float64{}
	for _, s := range subsets {
		present[setsKey(sorted(s.Sets))] = true
		if len(s.Sets) == 1 {
			sizes[s.Sets[0]] = s.Size
		}
	}
	var added []Subset
	empty := ""
	for _, s := range subsets {
		if len(s.Sets) < 3 {
			continue
		}
		members := sorted(s.Sets)
		for i := 0; i < len(members)-1; i++ {
			for j := i + 1; j < len(members); j++ {
				pair := []string{members[i], members[j]}
				key := setsKey(pair)
				if present[key] {
					continue
				}
				present[key] = true
				a, okA := sizes[pair[0]]
				b, okB := sizes[pair[1]]
				size := 2.5
				if okA && okB {
					size = min(a, b) / 4
				}
				added = append(added, Subset{Sets: pair, Size: size, Label: &empty})
			}
		}
	}
	if len(added) > 0 {
		return append(slices.Clip(subsets), added...)
	}
	return subsets
}

// sorted is a copy of ids sorted as JavaScript's sort orders strings.
func sorted(ids []string) []string {
	out := slices.Clone(ids)
	slices.SortStableFunc(out, compare16)
	return out
}

// lensArea is the area two circles share when their centers are d apart, as Pele's lensArea
// works it out.
func lensArea(r1, r2, d float64) float64 {
	if d >= r1+r2 {
		return 0
	}
	small := min(r1, r2)
	if d <= math.Abs(r1-r2) {
		return float64(math.Pi*small) * small
	}
	a1 := float64(r1 * r1 * jsmath.Acos((float64(d*d)+float64(r1*r1)-float64(r2*r2))/(2*d*r1)))
	a2 := float64(r2 * r2 * jsmath.Acos((float64(d*d)+float64(r2*r2)-float64(r1*r1))/(2*d*r2)))
	a3 := float64(0.5 * math.Sqrt(max((-d+r1+r2)*(d+r1-r2)*(d-r1+r2)*(d+r1+r2), 0)))
	return a1 + a2 - a3
}

// distanceForOverlap is how far apart two circles sit for their overlap to have the given
// area, as Pele's distanceForOverlap finds it: the area falls as the distance grows, so
// bisection finds it.
func distanceForOverlap(r1, r2, area float64) float64 {
	low := math.Abs(r1 - r2)
	high := r1 + r2
	if !(area > 0) {
		return high
	}
	m := min(r1, r2)
	if area >= math.Pi*float64(m*m) {
		return low
	}
	for range 48 {
		middle := (low + high) / 2
		if lensArea(r1, r2, middle) > area {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2
}

// margin is how far a point is from the nearest outline it must stay within or without, as
// Pele's margin gives it. It gives up, with some value below floor, as soon as the point
// cannot reach floor.
func margin(circles []circle, inside, outside []int, x, y, floor float64) float64 {
	best := math.Inf(1)
	for _, i := range inside {
		c := &circles[i]
		dx := x - c.x
		dy := y - c.y
		room := c.r - math.Sqrt(float64(dx*dx)+float64(dy*dy))
		if room < best {
			if room < floor {
				return room
			}
			best = room
		}
	}
	for _, i := range outside {
		c := &circles[i]
		dx := x - c.x
		dy := y - c.y
		room := math.Sqrt(float64(dx*dx)+float64(dy*dy)) - c.r
		if room < best {
			if room < floor {
				return room
			}
			best = room
		}
	}
	return best
}

// findSpot finds the roomiest point of a region by sampling the smallest of its circles on a
// grid and then walking uphill from the best sample, as Pele's findSpot does. Ties go to the
// point nearest the middle of the region, so a symmetric arrangement gets symmetric labels.
func findSpot(circles []circle, inside, outside []int) spot {
	cx, cy := 0.0, 0.0
	smallest := &circles[inside[0]]
	for _, i := range inside {
		c := &circles[i]
		cx += c.x / float64(len(inside))
		cy += c.y / float64(len(inside))
		if c.r < smallest.r {
			smallest = c
		}
	}
	r := smallest.r
	x, y := cx, cy
	best := margin(circles, inside, outside, x, y, math.Inf(-1))
	off := 0.0
	consider := func(px, py float64) bool {
		m := margin(circles, inside, outside, px, py, best-float64(r*1e-9))
		if !(m > best-float64(r*1e-9)) {
			return false
		}
		distance := jsmath.Hypot(px-cx, py-cy)
		if m > best+float64(r*1e-9) || distance < off-float64(r*1e-9) {
			best = m
			off = distance
			x, y = px, py
			return true
		}
		return false
	}
	const grid = 10
	for gy := -grid; gy <= grid; gy++ {
		for gx := -grid; gx <= grid; gx++ {
			if gx*gx+gy*gy <= grid*grid {
				consider(smallest.x+float64(gx)*r/grid, smallest.y+float64(gy)*r/grid)
			}
		}
	}
	step := r / grid
	for round := 0; round < 40 && step > r*1e-6; round++ {
		moved := false
		for k := range 8 {
			angle := float64(k) * tau / 8
			if consider(x+float64(jsmath.Cos(angle)*step), y+float64(jsmath.Sin(angle)*step)) {
				moved = true
			}
		}
		if !moved {
			step /= 2
		}
	}
	return spot{x, y, best}
}

// layoutCircles places one circle per set, as Pele's layoutCircles does, with areas in
// proportion to the sizes of the sets and each pair as far apart as its overlap asks: one,
// two and three sets exactly, more by relaxing a ring of circles toward those distances.
// Lengths are in units of the square root of a size.
func layoutCircles(subsets []Subset) []circle {
	// The sizes by id, in the order the ids came, as a Map keeps them.
	var ids []string
	sizes := map[string]float64{}
	for _, s := range subsets {
		for _, id := range s.Sets {
			if _, ok := sizes[id]; !ok {
				ids = append(ids, id)
				sizes[id] = 10
			}
		}
	}
	for _, s := range subsets {
		if len(s.Sets) == 1 {
			sizes[s.Sets[0]] = s.Size
		}
	}

	index := map[string]int{}
	var circles []circle
	largest := 0.0
	finite := func(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }
	for _, id := range ids {
		if size := sizes[id]; finite(size) && size > largest {
			largest = size
		}
	}
	if largest == 0 {
		largest = 10
	}
	widest := math.Sqrt(largest / math.Pi)
	for _, id := range ids {
		index[id] = len(circles)
		// A set too small to hold its name, or with no size at all, still gets a circle that
		// shows.
		r := 0.0
		if size := sizes[id]; finite(size) && size > 0 {
			r = math.Sqrt(size / math.Pi)
		}
		circles = append(circles, circle{id: id, r: max(r, widest*0.3)})
	}
	n := len(circles)
	if n < 2 {
		return circles
	}

	// The overlap each pair was given, by the indexes of its circles.
	overlaps := map[int]float64{}
	for _, s := range ensurePairwiseSubsets(subsets) {
		if len(s.Sets) != 2 {
			continue
		}
		a, b := index[s.Sets[0]], index[s.Sets[1]]
		if a != b {
			overlaps[min(a, b)*n+max(a, b)] = s.Size
		}
	}
	// How far apart each overlapping pair should be. Pairs that do not overlap only have to
	// keep a gap.
	wanted := map[int]float64{}
	for key, area := range overlaps {
		wanted[key] = distanceForOverlap(circles[key/n].r, circles[key%n].r, area)
	}
	gap := float64(widest * 0.15)
	distance := func(a, b int) float64 {
		if d, ok := wanted[a*n+b]; ok {
			return d
		}
		return circles[a].r + circles[b].r + gap
	}

	if n == 2 {
		circles[1].x = distance(0, 1)
		return circles
	}

	if n == 3 {
		// Two sets side by side and the third below, the corners of a triangle whose sides
		// are the three distances. An overlap of all three has to exist, so the triangle
		// shrinks until it does.
		ab, ac, bc := distance(0, 1), distance(0, 2), distance(1, 2)
		triple := false
		for _, s := range subsets {
			if distinct(s.Sets) == 3 {
				triple = true
				break
			}
		}
		for range 40 {
			x := 0.0
			if ab > 0 {
				x = (float64(ab*ab) + float64(ac*ac) - float64(bc*bc)) / (2 * ab)
			}
			circles[1].x = ab
			circles[2].x = max(-ac, min(ac, x))
			circles[2].y = math.Sqrt(max(float64(ac*ac)-float64(x*x), 0))
			if !triple || findSpot(circles, []int{0, 1, 2}, nil).margin > widest*0.12 {
				break
			}
			ab *= 0.94
			ac *= 0.94
			bc *= 0.94
		}
		return circles
	}

	total := 0.0
	for _, c := range circles {
		total += float64(2*c.r) + gap
	}
	rounds := min(300, 3_000_000/(n*n))
	if n <= 64 {
		ring := total / tau
		for i := range circles {
			angle := float64(i)*tau/float64(n) - math.Pi/2
			circles[i].x = float64(jsmath.Cos(angle) * ring)
			circles[i].y = float64(jsmath.Sin(angle) * ring)
		}
	} else {
		columns := int(math.Ceil(math.Sqrt(float64(n))))
		cell := float64(2*widest) + gap
		for i := range circles {
			circles[i].x = float64(float64(i%columns) * cell)
			circles[i].y = float64(float64(i/columns) * cell)
		}
	}
	dx := make([]float64, n)
	dy := make([]float64, n)
	pulls := make([]int, n)
	for range rounds {
		clear(dx)
		clear(dy)
		clear(pulls)
		for a := 0; a < n; a++ {
			for b := a + 1; b < n; b++ {
				overlap, overlaps := wanted[a*n+b]
				apart := circles[a].r + circles[b].r + gap
				ux := circles[b].x - circles[a].x
				uy := circles[b].y - circles[a].y
				length := jsmath.Hypot(ux, uy)
				if !overlaps && length >= apart {
					continue
				}
				if length < 1e-9 {
					ux = jsmath.Cos(float64(a + b))
					uy = jsmath.Sin(float64(a + b))
					length = 1
				}
				target := apart
				if overlaps {
					target = overlap
				}
				move := (length - target) / 2 / length
				dx[a] += float64(ux * move)
				dy[a] += float64(uy * move)
				dx[b] -= float64(ux * move)
				dy[b] -= float64(uy * move)
				pulls[a]++
				pulls[b]++
			}
		}
		for i := range n {
			if pulls[i] == 0 {
				continue
			}
			circles[i].x += dx[i] / float64(pulls[i])
			circles[i].y += dy[i] / float64(pulls[i])
		}
	}
	return circles
}

// distinct is the number of different ids in a list, the size of Pele's new Set of them.
func distinct(ids []string) int {
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	return len(seen)
}

// corner is a corner of a region: where the outlines of circles a and b cross.
type corner struct {
	x, y float64
	a, b int
}

// regionPath is the outline of the region all the given circles share, as Pele's regionPath
// writes it, or nil if they share none. Its sides are arcs, drawn with ArcTo as SVG's A
// command that Pele writes draws them.
func regionPath(circles []circle) *scene.Path {
	whole := func(c *circle) *scene.Path {
		return new(scene.Path).MoveTo(c.x-c.r, c.y).
			ArcTo(c.r, c.r, 0, true, true, c.x+c.r, c.y).
			ArcTo(c.r, c.r, 0, true, true, c.x-c.r, c.y).
			Close()
	}
	if len(circles) == 1 {
		return whole(&circles[0])
	}
	within := func(x, y float64, skip, other int) bool {
		for k := range circles {
			c := &circles[k]
			if k != skip && k != other && jsmath.Hypot(x-c.x, y-c.y) > c.r*(1+1e-9) {
				return false
			}
		}
		return true
	}
	// The corners of the region: where two outlines cross inside all the other circles.
	var corners []corner
	for i := range circles {
		for j := i + 1; j < len(circles); j++ {
			a, b := &circles[i], &circles[j]
			d := jsmath.Hypot(b.x-a.x, b.y-a.y)
			if d >= a.r+b.r || d <= math.Abs(a.r-b.r) || d == 0 {
				continue
			}
			along := (float64(a.r*a.r) - float64(b.r*b.r) + float64(d*d)) / (2 * d)
			up := math.Sqrt(max(float64(a.r*a.r)-float64(along*along), 0))
			mx := a.x + (b.x-a.x)*along/d
			my := a.y + (b.y-a.y)*along/d
			for _, sign := range [2]float64{1, -1} {
				x := mx + sign*up*(b.y-a.y)/d
				y := my - sign*up*(b.x-a.x)/d
				if within(x, y, i, j) {
					corners = append(corners, corner{x, y, i, j})
				}
			}
		}
	}
	if len(corners) < 2 {
		// No corners: either one circle lies inside all the others, or there is nothing
		// shared.
		smallest := 0
		for k := range circles {
			if circles[k].r < circles[smallest].r {
				smallest = k
			}
		}
		s := &circles[smallest]
		for k := range circles {
			c := &circles[k]
			if k != smallest && jsmath.Hypot(c.x-s.x, c.y-s.y)+s.r > c.r*(1+1e-9) {
				return nil
			}
		}
		return whole(s)
	}
	cx, cy := 0.0, 0.0
	for _, p := range corners {
		cx += p.x / float64(len(corners))
		cy += p.y / float64(len(corners))
	}
	// Pele sorts by the difference of the angles, which orders them as the angles do.
	slices.SortStableFunc(corners, func(p, q corner) int {
		d := jsmath.Atan2(p.y-cy, p.x-cx) - jsmath.Atan2(q.y-cy, q.x-cx)
		switch {
		case d < 0:
			return -1
		case d > 0:
			return 1
		}
		return 0
	})
	path := new(scene.Path)
	for i := range corners {
		from := corners[i]
		to := corners[(i+1)%len(corners)]
		if i == 0 {
			path.MoveTo(from.x, from.y)
		}
		// The side between two corners is an arc of a circle that passes through both; of
		// those, the one whose arc stays inside the others.
		arc := -1
		large := false
		narrowest := math.Inf(1)
		for _, k := range [2]int{from.a, from.b} {
			if k != to.a && k != to.b {
				continue
			}
			c := &circles[k]
			start := jsmath.Atan2(from.y-c.y, from.x-c.x)
			sweep := jsmath.Atan2(to.y-c.y, to.x-c.x) - start
			if sweep <= 0 {
				sweep += tau
			}
			mx := c.x + float64(c.r*jsmath.Cos(start+sweep/2))
			my := c.y + float64(c.r*jsmath.Sin(start+sweep/2))
			width := jsmath.Hypot(mx-(from.x+to.x)/2, my-(from.y+to.y)/2)
			if width < narrowest {
				narrowest = width
				arc = k
				large = sweep > math.Pi
			}
		}
		if arc == -1 {
			path.LineTo(to.x, to.y)
		} else {
			r := circles[arc].r
			path.ArcTo(r, r, 0, large, true, to.x, to.y)
		}
	}
	return path.Close()
}
