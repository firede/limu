// Command gen writes what the website shows of limu's own work, before Astro builds it: the
// mark and the icon, the SVG of every example, drawn by the svg package, and the theme's
// colors as the --limu-* variables the SVG is drawn in. Run it from the website directory:
//
//	go run ./gen
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/gallery"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/style"
	"github.com/firede/limu/svg"
)

// fontFamily is what the SVG sets its text in. Layout measures text with text.Arial, so the
// family is Arial's, or one as wide.
const fontFamily = "Arial, Helvetica, sans-serif"

// The widths a diagram is drawn at, as Pele's site draws its examples: about the width of the
// column of a page on a desktop, of a tablet, and of a phone. CSS shows the widest that the
// figure has room for.
const wideWidth, mediumWidth, narrowWidth = 640, 480, 320

// now is the present a gantt chart takes, so that a build draws the same SVG whenever it runs.
var now = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func main() {
	log.SetFlags(0)
	write := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			log.Fatal(err)
		}
	}
	write("public/mark.svg", svgFile(64, markSVG()))
	write("public/mark-16.svg", svgFile(16, mark16SVG()))
	write("public/icon.svg", svgFile(64, iconSVG()))
	write("public/icon-16.svg", svgFile(16, icon16SVG()))
	write("src/generated/icon-full.svg", svgFile(64, iconFullSVG()))
	write("src/generated/mark.json", mustJSON(map[string]string{"mark": markSVG(), "mark16": mark16SVG()}))
	write("src/generated/limu-theme.css", themeCSS())
	write("src/generated/home.json", mustJSON(home()))

	types, failed := examples()
	write("src/generated/examples.json", mustJSON(types))
	n := 0
	for _, t := range types {
		n += len(t.Examples)
	}
	fmt.Printf("gen: %d examples of %d types drawn", n, len(types))
	if failed > 0 {
		fmt.Printf(", %d left out as limu could not lay them out", failed)
	}
	fmt.Println()
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	return string(b) + "\n"
}

// A galleryType is a type of diagram and its examples, as the gallery keeps them, each with
// the drawing of it.
type galleryType struct {
	Type        string           `json:"type"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Beta        bool             `json:"beta"`
	Examples    []galleryExample `json:"examples"`
}

type galleryExample struct {
	Title    string `json:"title"`
	Source   string `json:"source"`
	Featured bool   `json:"featured,omitempty"`
	drawing
}

// A drawing is a diagram drawn at the three widths: Medium and Narrow are left empty where the
// diagram draws no narrower than at the width above.
type drawing struct {
	SVG    string `json:"svg"`
	Medium string `json:"medium,omitempty"`
	Narrow string `json:"narrow,omitempty"`
}

// draw lays a diagram out at the three widths and writes each as SVG, in Kohu, its colors
// written as the --limu-* variables, so that the page's theme decides them.
func draw(src string) (drawing, error) {
	d, err := limu.Parse(src, diagram.Options{})
	if err != nil {
		return drawing{}, err
	}
	o := &svg.Options{Look: style.Look{Theme: style.Kohu()}, FontFamily: fontFamily}
	var out drawing
	widest := 0.0
	for i, w := range []float64{wideWidth, mediumWidth, narrowWidth} {
		sc, err := d.Layout(&layout.Options{MaxWidth: w, Now: now, Location: time.UTC})
		if err != nil {
			return drawing{}, err
		}
		if i > 0 && sc.Width >= widest {
			continue
		}
		widest = sc.Width
		switch i {
		case 0:
			out.SVG = svg.Render(sc, o)
		case 1:
			out.Medium = svg.Render(sc, o)
		case 2:
			out.Narrow = svg.Render(sc, o)
		}
	}
	return out, nil
}

// examples draws each example of limu's gallery. It reports how many examples limu could
// not lay out, which it leaves out.
func examples() ([]galleryType, int) {
	var types []galleryType
	failed := 0
	for _, t := range gallery.All() {
		gt := galleryType{Type: string(t.Type), Title: t.Title, Description: t.Description, Beta: t.Beta}
		for _, e := range t.Examples {
			d, err := draw(e.Source)
			if err != nil {
				log.Printf("gen: %s, %q: %v", t.Type, e.Title, err)
				failed++
				continue
			}
			gt.Examples = append(gt.Examples, galleryExample{Title: e.Title, Source: e.Source, Featured: e.Featured, drawing: d})
		}
		types = append(types, gt)
	}
	return types, failed
}

// themeCSS writes Kohu and KohuDark as the --limu-* variables the svg package draws with:
// Kohu for the page's light theme and KohuDark for its dark one, chosen as base.scss chooses
// the site's own colors. An element with data-diagram-theme draws the diagrams in it in
// Kohu, KohuDark or Default, whatever the page's theme, as the examples' switch sets it.
func themeCSS() string {
	vars := func(th *style.Theme) string {
		var b strings.Builder
		put := func(name string, c style.Color) { fmt.Fprintf(&b, "  --limu-%s: %s;\n", name, c) }
		put("bg", th.Bg)
		put("fg", th.Fg)
		put("muted", th.Muted)
		put("line", th.Line)
		put("surface", th.Surface)
		put("surface-alt", th.SurfaceAlt)
		put("border", th.Border)
		put("accent", th.Accent)
		put("critical", th.Critical)
		for i, c := range th.Series {
			put(fmt.Sprintf("series-%d", i+1), c)
		}
		fmt.Fprintf(&b, "  --limu-font: %s;\n", fontFamily)
		return b.String()
	}
	light, dark := vars(style.Kohu()), vars(style.KohuDark())
	return "/* Written by gen from style.Kohu, style.KohuDark and style.Default; do not edit. */\n" +
		":root {\n" + light + "}\n" +
		"@media (prefers-color-scheme: dark) {\n  :root:not([data-theme=\"light\"]) {\n" + indent(dark) + "  }\n}\n" +
		":root[data-theme=\"dark\"] {\n" + dark + "}\n" +
		"[data-diagram-theme=\"kohu\"] {\n" + light + "}\n" +
		"[data-diagram-theme=\"kohu-dark\"] {\n" + dark + "}\n" +
		"[data-diagram-theme=\"default\"] {\n" + vars(style.Default()) + "}\n"
}

// indent indents each line of s by two spaces more.
func indent(s string) string {
	lines := strings.SplitAfter(s, "\n")
	var b strings.Builder
	for _, l := range lines {
		if l != "" {
			b.WriteString("  " + l)
		}
	}
	return b.String()
}
