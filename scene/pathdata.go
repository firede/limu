package scene

import (
	"fmt"
	"strconv"
)

// ParsePath reads path data, as SVG's d attribute holds it: the commands M, L, H, V, C, S,
// Q, T, A and Z, each also in lower case for numbers relative to where the path is, with
// numbers repeated for a command given once. Diagrams use it for the outlines Pele draws
// from path data of its own, as the icons architecture and tree view diagrams know.
// Data that does not follow SVG's grammar is an error.
func ParsePath(d string) (*Path, error) {
	r := pathReader{s: d}
	p := new(Path)
	var cmd byte
	// The control point a smooth curve reflects, and the kind of curve that set it.
	var ctrl Point
	var ctrlOf byte
	r.space()
	for r.i < len(r.s) {
		if c := r.s[r.i]; isPathCommand(c) {
			cmd = c
			r.i++
		} else if cmd == 0 || cmd == 'Z' || cmd == 'z' {
			return nil, fmt.Errorf("path data: number without a command at %d", r.i)
		} else if cmd == 'M' {
			cmd = 'L' // coordinates after a moveto's first are lines
		} else if cmd == 'm' {
			cmd = 'l'
		}
		at := p.Last()
		rel := cmd >= 'a'
		var ox, oy float64
		if rel {
			ox, oy = at.X, at.Y
		}
		var n [7]float64
		kind := cmd &^ 0x20
		for k := range pathArgs(kind) {
			r.comma(k > 0)
			var err error
			if kind == 'A' && (k == 3 || k == 4) {
				n[k], err = r.flag()
			} else {
				n[k], err = r.number()
			}
			if err != nil {
				return nil, err
			}
		}
		switch kind {
		case 'M':
			p.MoveTo(ox+n[0], oy+n[1])
		case 'L':
			p.LineTo(ox+n[0], oy+n[1])
		case 'H':
			p.LineTo(ox+n[0], at.Y)
		case 'V':
			p.LineTo(at.X, oy+n[0])
		case 'C':
			p.CubeTo(ox+n[0], oy+n[1], ox+n[2], oy+n[3], ox+n[4], oy+n[5])
			ctrl = Point{ox + n[2], oy + n[3]}
		case 'S':
			c1 := at
			if ctrlOf == 'C' {
				c1 = Point{2*at.X - ctrl.X, 2*at.Y - ctrl.Y}
			}
			p.CubeTo(c1.X, c1.Y, ox+n[0], oy+n[1], ox+n[2], oy+n[3])
			ctrl, kind = Point{ox + n[0], oy + n[1]}, 'C'
		case 'Q':
			p.QuadTo(ox+n[0], oy+n[1], ox+n[2], oy+n[3])
			ctrl = Point{ox + n[0], oy + n[1]}
		case 'T':
			c := at
			if ctrlOf == 'Q' {
				c = Point{2*at.X - ctrl.X, 2*at.Y - ctrl.Y}
			}
			p.QuadTo(c.X, c.Y, ox+n[0], oy+n[1])
			ctrl, kind = c, 'Q'
		case 'A':
			if len(p.Ops) == 0 {
				p.MoveTo(0, 0)
			}
			p.ArcTo(n[0], n[1], n[2], n[3] != 0, n[4] != 0, ox+n[5], oy+n[6])
		case 'Z':
			p.Close()
		}
		ctrlOf = kind
		r.space()
		if r.i < len(r.s) && r.s[r.i] == ',' {
			if kind == 'Z' {
				return nil, fmt.Errorf("path data: comma after a closepath at %d", r.i)
			}
			r.i++
			r.space()
		}
	}
	return p, nil
}

// Scaled is the path scaled by s about the origin, then moved by dx, dy, as an icon drawn
// on a grid of its own is placed in a box.
func (p *Path) Scaled(s, dx, dy float64) *Path {
	q := &Path{Ops: append([]Op(nil), p.Ops...), Args: make([]float64, len(p.Args))}
	for i, v := range p.Args {
		if i%2 == 0 {
			q.Args[i] = v*s + dx
		} else {
			q.Args[i] = v*s + dy
		}
	}
	return q
}

func isPathCommand(c byte) bool { return pathArgs(c&^0x20) >= 0 }

// pathArgs is how many numbers a command in upper case takes, or -1 for no command.
func pathArgs(c byte) int {
	switch c {
	case 'Z':
		return 0
	case 'H', 'V':
		return 1
	case 'M', 'L', 'T':
		return 2
	case 'S', 'Q':
		return 4
	case 'C':
		return 6
	case 'A':
		return 7
	}
	return -1
}

type pathReader struct {
	s string
	i int
}

func (r *pathReader) space() {
	for r.i < len(r.s) {
		switch r.s[r.i] {
		case ' ', '\t', '\n', '\r', '\f':
			r.i++
		default:
			return
		}
	}
}

// comma skips the space, and between numbers a comma, that may come before a number.
func (r *pathReader) comma(between bool) {
	r.space()
	if between && r.i < len(r.s) && r.s[r.i] == ',' {
		r.i++
		r.space()
	}
}

// number reads a number as SVG writes one: a sign, digits with a point among or before
// them, and an exponent. It ends where the next character can't continue it, so "1-2" is
// two numbers and ".5.5" is two too.
func (r *pathReader) number() (float64, error) {
	from, i := r.i, r.i
	digits := func() int {
		n := 0
		for i < len(r.s) && r.s[i] >= '0' && r.s[i] <= '9' {
			i, n = i+1, n+1
		}
		return n
	}
	if i < len(r.s) && (r.s[i] == '+' || r.s[i] == '-') {
		i++
	}
	n := digits()
	if i < len(r.s) && r.s[i] == '.' {
		i++
		n += digits()
	}
	if n == 0 {
		return 0, fmt.Errorf("path data: number expected at %d", from)
	}
	if i < len(r.s) && (r.s[i] == 'e' || r.s[i] == 'E') {
		j := i
		i++
		if i < len(r.s) && (r.s[i] == '+' || r.s[i] == '-') {
			i++
		}
		if digits() == 0 {
			i = j // an e that no digits follow ends the number before it
		}
	}
	v, err := strconv.ParseFloat(r.s[from:i], 64)
	if err != nil {
		return 0, fmt.Errorf("path data: %v", err)
	}
	r.i = i
	return v, nil
}

// flag reads an arc's flag, a single 0 or 1 that needs nothing to part it from what follows.
func (r *pathReader) flag() (float64, error) {
	if r.i < len(r.s) && (r.s[r.i] == '0' || r.s[r.i] == '1') {
		r.i++
		return float64(r.s[r.i-1] - '0'), nil
	}
	return 0, fmt.Errorf("path data: flag expected at %d", r.i)
}
