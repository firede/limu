package common

import (
	"math"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// WithTitle puts a title above a scene that was laid out without one, as Pele's withTitle
// does: the scene grows to hold it, what was in it moves down below it, and across to stay
// centered where the title is the wider. The title is an item of the role given.
func WithTitle(sc *scene.Scene, title string, role scene.Role, measure text.Measurer, size float64) {
	label := text.Layout(title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold})
	if label.Height == 0 {
		return
	}
	shift := label.Height + 12
	full := max(sc.Width, math.Ceil(label.Width+2*Padding))
	dx := (full - sc.Width) / 2
	for i := range sc.Items {
		sc.Items[i].Move(dx, shift)
	}
	sc.Width = full
	sc.Height = math.Ceil(sc.Height + shift)
	sc.Items = append([]scene.Item{{
		Kind: scene.Text, Role: role, Label: &label,
		At:    scene.Point{X: full / 2, Y: Padding + label.Height/2},
		Style: style.Style{Text: style.Fg},
	}}, sc.Items...)
}
