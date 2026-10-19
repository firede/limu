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
// out again. Only what changes the size of things — the font, its sizes — is given to layout.
//
// Much of limu is ported from Pele, the Mermaid renderer of the Obsidian team, and its tests
// compare what limu does with what Pele does, number for number.
package limu
