// Package limu reads Mermaid diagrams and lays them out, in Go, for programs that draw them
// themselves: on a GPU, in a terminal, as SVG.
//
// A diagram goes through three stages, each a package of its own:
//
//   - parsing reads the diagram's text into a model of it;
//   - layout places the model's parts, measuring text with a text.Measurer the host gives,
//     into a scene.Scene: what each part is and where it is, but not how it looks;
//   - a backend draws the scene, settling its colors from a style.Look: the svg package
//     writes SVG, and others draw on screens of their own.
//
// Style is not layout: a scene is drawn in any theme, with any overrides, without being laid
// out again. Only what changes where things are and how large they are is given to layout, in
// a layout.Options: the measurer and the size of text, the width the host has and whether a
// diagram may turn to fit it, the padding about it, a grid to snap to, how edges are routed,
// and the present and the time zone of a gantt chart.
//
// Layout reads a diagram's text and lays it out in one call. A host that lays the same
// diagram out more than once, as when the width it has changes, parses it once with Parse,
// which takes the limits and the configuration Pele's parse takes, and lays the Diagram out
// with its Layout method as often as it needs, from as many goroutines as it likes.
//
// Much of limu is ported from Pele, the Mermaid renderer of the Obsidian team, and its tests
// compare what limu does with what Pele does, number for number.
package limu
