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

	"github.com/firede/limu"
	"github.com/firede/limu/style"
	"github.com/firede/limu/svg"
)

// gallery is where the examples are kept, with their titles and descriptions: the
// playground's, as the website and the playground show the same.
const gallery = "../limumygo/examples/playground/internal/gallery/examples.json"

// fontFamily is what the SVG sets its text in. Layout measures text with text.Arial, so the
// family is Arial's, or one as wide.
const fontFamily = "Arial, Helvetica, sans-serif"

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
// the SVG it is drawn as.
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
	SVG      string `json:"svg"`
}

// examples reads the gallery and draws each example in Kohu, its colors written as the
// --limu-* variables, so that the page's theme decides them. It reports how many examples
// limu could not lay out, which it leaves out.
func examples() ([]galleryType, int) {
	b, err := os.ReadFile(gallery)
	if err != nil {
		log.Fatal(err)
	}
	var g struct{ Types []galleryType }
	if err := json.Unmarshal(b, &g); err != nil {
		log.Fatal(err)
	}
	o := &svg.Options{Look: style.Look{Theme: style.Kohu()}, FontFamily: fontFamily}
	failed := 0
	for i := range g.Types {
		t := &g.Types[i]
		kept := t.Examples[:0]
		for _, e := range t.Examples {
			sc, err := limu.Layout(e.Source, nil)
			if err != nil {
				log.Printf("gen: %s, %q: %v", t.Type, e.Title, err)
				failed++
				continue
			}
			e.SVG = svg.Render(sc, o)
			kept = append(kept, e)
		}
		t.Examples = kept
	}
	return g.Types, failed
}

// themeCSS writes Kohu and KohuDark as the --limu-* variables the svg package draws with:
// Kohu for the page's light theme and KohuDark for its dark one, chosen as base.scss chooses
// the site's own colors.
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
	return "/* Written by gen from style.Kohu and style.KohuDark; do not edit. */\n" +
		":root {\n" + light + "}\n" +
		"@media (prefers-color-scheme: dark) {\n  :root:not([data-theme=\"light\"]) {\n" + indent(dark) + "  }\n}\n" +
		":root[data-theme=\"dark\"] {\n" + dark + "}\n"
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
