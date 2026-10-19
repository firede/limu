package scene

import (
	"slices"
	"testing"
)

func TestMove(t *testing.T) {
	path := new(Path).MoveTo(1, 2).LineTo(3, 4)
	route := []Point{{1, 1}, {2, 2}}
	pic := &Picture{Box: Rect{1, 1, 2, 2}}
	it := Item{
		Box:     Rect{0, 0, 10, 10},
		Parts:   []Part{{Path: path}, {Box: Rect{1, 1, 2, 2}}},
		Route:   route,
		Ends:    [2]Marker{{At: Point{5, 5}}},
		At:      Point{5, 5},
		Picture: pic,
	}
	shared := it
	it.Move(10, 20)
	if it.Box != (Rect{10, 20, 10, 10}) || it.At != (Point{15, 25}) || it.Ends[0].At != (Point{15, 25}) {
		t.Errorf("box, at or end not moved: %v %v %v", it.Box, it.At, it.Ends[0].At)
	}
	if !slices.Equal(it.Parts[0].Path.Args, []float64{11, 22, 13, 24}) || it.Parts[1].Box != (Rect{11, 21, 2, 2}) {
		t.Errorf("parts not moved: %v %v", it.Parts[0].Path.Args, it.Parts[1].Box)
	}
	if !slices.Equal(it.Route, []Point{{11, 21}, {12, 22}}) || it.Picture.Box != (Rect{11, 21, 2, 2}) {
		t.Errorf("route or picture not moved: %v %v", it.Route, it.Picture.Box)
	}
	// What the item shared before it moved stays where it was.
	if !slices.Equal(path.Args, []float64{1, 2, 3, 4}) || shared.Parts[1].Box != (Rect{1, 1, 2, 2}) ||
		route[0] != (Point{1, 1}) || pic.Box != (Rect{1, 1, 2, 2}) {
		t.Errorf("shared parts moved: %v %v %v %v", path.Args, shared.Parts[1].Box, route[0], pic.Box)
	}
}
