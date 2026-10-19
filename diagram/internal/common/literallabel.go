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
