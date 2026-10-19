package main

import "testing"

func TestFormatSVG(t *testing.T) {
	doc := `<svg a="1"><g class="x>y"><rect x="0"/><text xml:space="preserve"><tspan>a  b</tspan></text></g></svg>`
	want := `<svg a="1">
  <g class="x>y">
    <rect x="0"/>
    <text xml:space="preserve"><tspan>a  b</tspan></text>
  </g>
</svg>`
	if got := formatSVG(doc); got != want {
		t.Errorf("formatSVG:\n%s\nwant\n%s", got, want)
	}
	for _, bad := range []string{`<svg><g></svg>`, `<svg></g></svg>`, `<svg a="1`, `<svg><text>a</svg>`} {
		if got := formatSVG(bad); got != bad {
			t.Errorf("formatSVG(%q) = %q, want it as it was", bad, got)
		}
	}
}
