package scene

import (
	"reflect"
	"testing"
)

func TestParsePath(t *testing.T) {
	for _, c := range []struct {
		d    string
		want *Path
	}{
		{"M3 4h18v7H3z", new(Path).MoveTo(3, 4).LineTo(21, 4).LineTo(21, 11).LineTo(3, 11).Close()},
		// Numbers repeated after a moveto are lines; a sign or a second point parts numbers.
		{"m1,2 3-4 .5.5", new(Path).MoveTo(1, 2).LineTo(4, -2).LineTo(4.5, -1.5)},
		{"M7 7.5h.01", new(Path).MoveTo(7, 7.5).LineTo(7.01, 7.5)},
		// After a closepath, relative numbers start from the figure's start.
		{"M1 1L5 1 5 5Zl1 0", new(Path).MoveTo(1, 1).LineTo(5, 1).LineTo(5, 5).Close().LineTo(2, 1)},
		{"M0 0C1 0 2 1 2 2S3 4 4 4", new(Path).MoveTo(0, 0).CubeTo(1, 0, 2, 1, 2, 2).CubeTo(2, 3, 3, 4, 4, 4)},
		{"M0 0Q1 1 2 0T4 0", new(Path).MoveTo(0, 0).QuadTo(1, 1, 2, 0).QuadTo(3, -1, 4, 0)},
		{"M0 0S1 1 2 2", new(Path).MoveTo(0, 0).CubeTo(0, 0, 1, 1, 2, 2)},
		{"M1e1 2E-1", new(Path).MoveTo(10, 0.2)},
		// Flags need nothing between them.
		{"M10 0a10 10 0 0110-10", new(Path).MoveTo(10, 0).ArcTo(10, 10, 0, false, true, 20, -10)},
		{"M4 6a8 3 0 1 0 16 0", new(Path).MoveTo(4, 6).ArcTo(8, 3, 0, true, false, 20, 6)},
	} {
		got, err := ParsePath(c.d)
		if err != nil {
			t.Errorf("%q: %v", c.d, err)
		} else if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: %v, want %v", c.d, got, c.want)
		}
	}
	for _, d := range []string{"1 2", "M1", "M1 2z3", "M1 2 x", "M0 0A1 1 0 2 0 1 1", "M1 2z,"} {
		if _, err := ParsePath(d); err == nil {
			t.Errorf("%q: no error", d)
		}
	}
}

func TestScaled(t *testing.T) {
	p := new(Path).MoveTo(0, 0).LineTo(24, 12).Close().Scaled(0.5, 10, 20)
	if want := new(Path).MoveTo(10, 20).LineTo(22, 26).Close(); !reflect.DeepEqual(p, want) {
		t.Errorf("%v", p)
	}
}
