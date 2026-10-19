package common

import (
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/text"
)

// LiteralLabel lays out text that must keep every character but the spaces its lines break
// at, as a filename or a field of a protocol, as Pele's literalLabel does: a line breaks at
// the last space that lets it fit room, or, in a word longer than that, between any two
// characters. Nothing in it is read as markup or entities.
func LiteralLabel(s string, measure text.Measurer, size, room float64, face text.Face) text.Label {
	var lines []string
	for _, raw := range strings.Split(s, "\n") {
		runes := []rune(raw)
		if len(runes) == 0 {
			lines = append(lines, "")
		}
		from := 0
		for from < len(runes) {
			fits := func(length int) bool {
				return measure.Width(string(runes[from:from+length]), size, face) <= room
			}
			hi := 1
			for hi < len(runes)-from && fits(hi) {
				hi = min(len(runes)-from, hi*2)
			}
			lo := 1
			for lo < hi {
				mid := (lo + hi + 1) / 2
				if fits(mid) {
					lo = mid
				} else {
					hi = mid - 1
				}
			}
			take := lo
			wraps := from+take < len(runes)
			if wraps && !chars.IsSpaceRune(runes[from+take]) {
				for i := take - 1; i > 0; i-- {
					if chars.IsSpaceRune(runes[from+i]) {
						take = i + 1
						break
					}
				}
			}
			line := string(runes[from : from+take])
			if wraps {
				if trimmed := chars.TrimEnd(line); trimmed != "" {
					line = trimmed
				}
			}
			lines = append(lines, line)
			from += take
			for wraps && from < len(runes) && chars.IsSpaceRune(runes[from]) {
				from++
			}
		}
	}
	if s == "" {
		lines = nil
	}
	l := text.Label{Size: size, LineHeight: jsmath.Round(size * text.LineHeight)}
	for _, line := range lines {
		w := measure.Width(line, size, face)
		l.Lines = append(l.Lines, []text.Span{{Text: line, Face: face, Width: w}})
		l.Widths = append(l.Widths, w)
		l.Width = max(l.Width, w)
	}
	l.Height = float64(len(lines)) * l.LineHeight
	return l
}

// BreakLabel breaks each line of a label that is wider than room between any two
// characters, so that no line but one holding a single character or an icon is wider, as
// Pele's breakLabel does. Lines that fit are kept as they are.
func BreakLabel(label text.Label, measure text.Measurer, room float64) text.Label {
	if label.Width <= room {
		return label
	}
	var lines [][]text.Span
	var widths []float64
	for i, line := range label.Lines {
		if label.Widths[i] <= room {
			lines = append(lines, line)
			widths = append(widths, label.Widths[i])
			continue
		}
		var current []text.Span
		width := 0.0
		commit := func() {
			lines = append(lines, current)
			widths = append(widths, width)
			current = nil
			width = 0
		}
		for _, span := range line {
			if span.Icon != "" {
				if len(current) > 0 && width+span.Width > room {
					commit()
				}
				current = append(current, span)
				width += span.Width
				continue
			}
			for _, r := range span.Text {
				char := string(r)
				var last *text.Span
				if n := len(current); n > 0 {
					last = &current[n-1]
				}
				joined := last != nil && last.Icon == "" && last.Face == span.Face
				var added float64
				if joined {
					added = measure.Width(last.Text+char, label.Size, span.Face) - last.Width
				} else {
					added = measure.Width(char, label.Size, span.Face)
				}
				if len(current) > 0 && width+added > room {
					commit()
					last = nil
					added = measure.Width(char, label.Size, span.Face)
				}
				if last != nil && last.Icon == "" && last.Face == span.Face {
					last.Text += char
					last.Width += added
				} else {
					current = append(current, text.Span{Text: char, Face: span.Face, Width: added})
				}
				width += added
			}
		}
		if len(current) > 0 || len(line) == 0 {
			commit()
		}
	}
	widest := 0.0
	for _, w := range widths {
		widest = max(widest, w)
	}
	label.Lines, label.Widths, label.Width = lines, widths, widest
	label.Height = float64(len(lines)) * label.LineHeight
	return label
}
