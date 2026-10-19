package main

import "github.com/egoist/mygo/ui"

// segmented is a segmented control of labels, of which *selected is the one chosen, drawn
// as the website's: a pill behind the segment chosen, on a track of the panel's color.
// Its Track's Changed reports a new choice.
func segmented(c *ui.Context, ap *appearance, selected *int, labels ...string) ui.Element {
	const pill = 999
	parts := ui.SegmentedBase(c, selected, len(labels))
	track := parts.Track
	track.Radius(pill).Background(ap.Track)
	track.Children(func() {
		for i, label := range labels {
			seg := parts.Segment(i).Padding(3, 12).Radius(pill)
			if i == *selected {
				seg.Background(ap.Knob)
			}
			seg.Children(func() {
				// The labels not chosen are muted, rather than the chosen one bold, so that
				// a choice does not change the segments' widths.
				txt := ui.Text(c, label).SingleLine().TextColor(ap.UI.TextMuted)
				if i == *selected || seg.Hovered() {
					txt.TextColor(ap.Strong)
				}
			})
		}
	})
	return track
}
