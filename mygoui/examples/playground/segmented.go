package main

import "github.com/egoist/mygo/ui"

// segmented is a segmented control of labels, of which *selected is the one chosen, drawn
// as Primer's: MyGo's own Segmented marks the chosen segment with the theme's pressed
// color, which in GitHub's dark themes is too near the track to tell. Its Track's Changed
// reports a new choice.
func segmented(c *ui.Context, ap *appearance, selected *int, labels ...string) ui.Element {
	t := ap.UI
	parts := ui.SegmentedBase(c, selected, len(labels))
	track := parts.Track
	track.Padding(2).Gap(2).Radius(t.Radius).Background(ap.Track).Border(1, t.Border)
	track.Children(func() {
		for i, label := range labels {
			seg := parts.Segment(i).Padding(3, 12).Radius(t.Radius-1).Border(1, ap.Track)
			if i == *selected {
				seg.Background(ap.Knob).Border(1, ap.KnobBorder)
			} else if seg.Hovered() {
				seg.Background(t.SurfaceHover)
			}
			seg.Children(func() {
				// The labels not chosen are muted, rather than the chosen one bold, so that
				// a choice does not change the segments' widths.
				txt := ui.Text(c, label).SingleLine()
				if i != *selected {
					txt.TextColor(t.TextMuted)
				}
			})
		}
	})
	return track
}
