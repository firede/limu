package common

import (
	"math"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// Fitted is one line of text fitted into a width, Pele's Fitted: the text, perhaps cut short,
// the size of type it is written in, and its width.
type Fitted struct {
	Text  string
	Size  float64
	Width float64
}

// FitText fits one line of text into a width, as Pele's fitText does: first by shrinking the
// type down to least, then by cutting the text and adding an ellipsis. It reports false when
// not even one character fits.
func FitText(s string, room, size, least float64, measure text.Measurer, face text.Face) (Fitted, bool) {
	if s == "" || room <= 0 {
		return Fitted{}, false
	}
	width := measure.Width(s, size, face)
	if width <= room {
		return Fitted{s, size, width}, true
	}

	// Width grows with the size of type, so one division lands on the right size or just
	// above it.
	for fitted := max(least, math.Floor(size*room/width)); fitted >= least; fitted-- {
		width = measure.Width(s, fitted, face)
		if width <= room {
			return Fitted{s, fitted, width}, true
		}
	}

	runes := []rune(s)
	cut := func(n int) string { return chars.TrimEnd(string(runes[:n])) + "…" }
	low, high := 0, len(runes)-1
	for low < high {
		mid := (low + high + 1) >> 1
		if measure.Width(cut(mid), least, face) <= room {
			low = mid
		} else {
			high = mid - 1
		}
	}
	if low == 0 {
		return Fitted{}, false
	}
	short := cut(low)
	width = measure.Width(short, least, face)
	if width > room {
		return Fitted{}, false
	}
	return Fitted{short, least, width}, true
}
