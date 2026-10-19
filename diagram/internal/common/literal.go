package common

import "github.com/firede/limu/text"

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
