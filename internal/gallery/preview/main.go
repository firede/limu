// Command preview draws the examples of the types it is given as one HTML page, to look at
// while writing them: each example's title and source, and its diagram laid out at most 640,
// 480 and 320 wide, as the website draws it, at its own size, in Kohu or, with -dark, in
// KohuDark. It reads each type's file from the directory as it is, so that a file being
// written in another does not stop it. Run it from limu's root:
//
//	go run ./internal/gallery/preview flowchart sequence > preview.html
package main

import (
	"flag"
	"fmt"
	"html"
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

func main() {
	log.SetFlags(0)
	dark := flag.Bool("dark", false, "draw in KohuDark")
	flag.Parse()
	theme, bg, fg := style.Kohu(), "#f1f6f5", "#0f262a"
	if *dark {
		theme, bg, fg = style.KohuDark(), "#0c1618", "#d0dddc"
	}
	o := &svg.Options{Look: style.Look{Theme: theme}, Literal: true, FontFamily: "Arial, Helvetica, sans-serif"}
	var b strings.Builder
	fmt.Fprintf(&b, `<!doctype html><meta charset="utf-8"><style>body{margin:24px;background:%s;color:%s;font:14px system-ui}h1{font-size:22px;margin:40px 0 4px}h2{font-size:16px;margin:32px 0 8px}pre{font-size:12px;opacity:.75}.row{display:flex;gap:24px;align-items:flex-start;flex-wrap:wrap}.w{border:1px dashed #8aa3a5;padding:4px}.w p{margin:0 0 4px;font-size:11px;opacity:.7}.err{color:#b0611a}</style>`, bg, fg)
	for _, name := range flag.Args() {
		text, err := os.ReadFile(filepath.Join("internal/gallery/examples", name+".md"))
		if err != nil {
			log.Fatal(err)
		}
		t, err := gallery.Parse(name, string(text))
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		fmt.Fprintf(&b, "<h1>%s</h1><p>%s</p>", html.EscapeString(t.Title), html.EscapeString(t.Description))
		for _, ex := range t.Examples {
			featured := ""
			if ex.Featured {
				featured = " (featured)"
			}
			fmt.Fprintf(&b, "<h2>%s%s</h2><pre>%s</pre><div class=row>", html.EscapeString(ex.Title), featured, html.EscapeString(ex.Source))
			d, err := limu.Parse(ex.Source, diagram.Options{})
			if err != nil {
				fmt.Fprintf(&b, "<p class=err>%s</p></div>", html.EscapeString(err.Error()))
				continue
			}
			for _, w := range []float64{640, 480, 320} {
				sc, err := d.Layout(&layout.Options{MaxWidth: w, Now: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), Location: time.UTC})
				if err != nil {
					fmt.Fprintf(&b, "<p class=err>%v: %s</p>", w, html.EscapeString(err.Error()))
					continue
				}
				fmt.Fprintf(&b, "<div class=w><p>at most %v: %.0f × %.0f</p>%s</div>", w, sc.Width, sc.Height, svg.Render(sc, o))
			}
			b.WriteString("</div>")
		}
	}
	os.Stdout.WriteString(b.String())
}
