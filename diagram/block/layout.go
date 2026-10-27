package block

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// A cell is a block in the grid of its composite, Pele's Cell: what it holds, how many
// columns it spans, the size it needs, and where it is placed.
type cell struct {
	children []*cell
	// columns is the grid width of a container: a positive count, or -1 to put every child
	// on one row.
	columns float64
	span    float64
	space   bool
	// head is the height kept above a container's grid for its title.
	head float64
	// nw and nh are the size the content needs: set by the caller for leaves, worked out
	// here for containers.
	nw, nh float64
	// Where the cell is placed: its place in the grid, then its center and size.
	col, row, cols float64
	heights        []float64
	// rowGaps and colGaps are extra room after each row or column of a container, which
	// edge labels that do not fit between its blocks take.
	rowGaps, colGaps []float64
	x, y, w, h       float64
}

// maxSpan is the most columns a block may span, Pele's MAX_SPAN.
const maxSpan = 1000

// calculateBlockPosition is the column and row of the cell at position in a grid of columns,
// as Pele's calculateBlockPosition gives them.
func calculateBlockPosition(columns, position float64) (px, py float64, err error) {
	if columns == 0 || !isInteger(columns) {
		return 0, 0, common.NewError("Columns must be an integer !== 0.", diagram.Semantic, "block")
	}
	if position < 0 || !isInteger(position) {
		return 0, 0, common.NewError("Position must be a non-negative integer."+chars.NumberString(position), diagram.Semantic, "block")
	}
	if columns < 0 {
		return position, 0, nil
	}
	return math.Mod(position, columns), math.Floor(position / columns), nil
}

// isInteger is Number.isInteger.
func isInteger(x float64) bool { return !math.IsInf(x, 0) && !math.IsNaN(x) && x == math.Trunc(x) }

// measure puts each child of a container in its cell of the grid and sizes the container, as
// Pele's measure does. Every column is as wide as the widest block needs. Every row that holds
// a plain block is as tall as the tallest block in the container, as in Mermaid; a row of
// nothing but containers is only as tall as they need.
func measure(box *cell, gap, inset float64) error {
	columns := box.columns
	var heights []float64
	position, cols, cw, tallest := 0.0, 0.0, 0.0, 0.0
	for _, child := range box.children {
		px, py, err := calculateBlockPosition(columns, position)
		if err != nil {
			return err
		}
		// Mermaid lets a block wider than the rest of its row stick out of the grid. Here it
		// stops at the last column.
		span := 1.0
		if child.span >= 1 {
			span = min(child.span, maxSpan)
		}
		if columns > 0 && span > columns-px {
			span = columns - px
		}
		child.span = span
		child.col = px
		child.row = py
		position += span
		if px+span > cols {
			cols = px + span
		}
		row := int(py)
		if row == len(heights) {
			heights = append(heights, 0)
		}
		if len(child.children) == 0 {
			heights[row] = -1
		} else if heights[row] >= 0 && child.nh > heights[row] {
			heights[row] = child.nh
		}
		if child.space {
			continue
		}
		width := (child.nw - float64(gap*(span-1))) / span
		if width > cw {
			cw = width
		}
		if child.nh > tallest {
			tallest = child.nh
		}
	}
	height := float64(float64(len(heights)-1)*gap) + total(box.rowGaps)
	for row := range heights {
		if heights[row] < 0 {
			heights[row] = tallest
		}
		height += heights[row]
	}
	box.cols = cols
	box.heights = heights
	box.nw = max(box.nw, float64(cols*cw)+float64((cols-1)*gap)+total(box.colGaps)+2*inset)
	box.nh = height + 2*inset + box.head
	return nil
}

// arrange gives each child of a container its cell, sharing out any room the container has
// beyond what it needs, as Pele's arrange does.
func arrange(box *cell, gap, inset float64) {
	cols, heights, rowGaps, colGaps := box.cols, box.heights, box.rowGaps, box.colGaps
	rows := float64(len(heights))
	cw := (box.w - 2*inset - float64((cols-1)*gap) - total(colGaps)) / cols
	room := box.h - 2*inset - box.head - float64((rows-1)*gap) - total(rowGaps)
	// offsets are the gaps before each column.
	var offsets []float64
	if len(colGaps) > 0 {
		sum := 0.0
		for col := 0; float64(col) < cols; col++ {
			offsets = append(offsets, sum)
			sum += at(colGaps, col)
		}
	}
	needed := 0.0
	for _, height := range heights {
		needed += height
	}
	scale, share := 0.0, 0.0
	if needed > 0 {
		scale = room / needed
	} else {
		share = room / rows
	}
	left := box.x - box.w/2 + inset
	top := box.y - box.h/2 + inset + box.head
	row := 0
	for _, child := range box.children {
		for ; float64(row) < child.row; row++ {
			top += float64(heights[row]*scale) + share + gap + at(rowGaps, row)
		}
		before := at(offsets, int(child.col))
		child.w = float64(cw*child.span) + float64(gap*(child.span-1)) + at(offsets, int(child.col+child.span-1)) - before
		child.h = float64(heights[row]*scale) + share
		child.x = left + float64(child.col*(cw+gap)) + before + child.w/2
		child.y = top + child.h/2
	}
}

// total is the sum of gaps, as Pele's total takes it.
func total(gaps []float64) float64 {
	sum := 0.0
	for _, gap := range gaps {
		sum += gap
	}
	return sum
}

// at is gaps[i], or 0 past their end, as JavaScript's gaps[i] ?? 0 reads a sparse array.
func at(gaps []float64, i int) float64 {
	if i < 0 || i >= len(gaps) {
		return 0
	}
	return gaps[i]
}

// layoutCells lays out a tree of cells in place, as Pele's layout does, and gives the size of
// the whole. The root has no border, so its grid starts at the origin.
func layoutCells(root *cell, gap float64) (width, height float64, err error) {
	order := []*cell{root}
	for i := 0; i < len(order); i++ {
		order = append(order, order[i].children...)
	}
	inset := func(i int) float64 {
		if i == 0 {
			return 0
		}
		return gap
	}
	for i := len(order) - 1; i >= 0; i-- {
		if box := order[i]; len(box.children) > 0 {
			if err := measure(box, gap, inset(i)); err != nil {
				return 0, 0, err
			}
		}
	}
	root.w = root.nw
	root.h = root.nh
	root.x = root.w / 2
	root.y = root.h / 2
	for i, box := range order {
		if len(box.children) > 0 {
			arrange(box, gap, inset(i))
		}
	}
	return root.w, root.h, nil
}
