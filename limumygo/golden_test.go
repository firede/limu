package limumygo

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/limumygo/internal/samples"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

var update = flag.Bool("update", false, "write the golden images anew")

// goFonts are the Go fonts, which the tests register, so that what they draw does not hang
// on the fonts of the system; how the system's text engine draws them still does, and the
// golden images are of one system each.
var goFonts = NewFonts("Go", "Go Mono")

func TestMain(m *testing.M) {
	for _, f := range [][]byte{goregular.TTF, gobold.TTF, goitalic.TTF, gomono.TTF} {
		if err := ui.RegisterFont(f, ""); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// snap renders a scene with View in a window of its size on the theme's background, at a
// scale, and compares it, pixel by pixel, with testdata/golden/name-GOOS.png. With no
// golden image for the system, it is skipped, unless -update writes one.
func snap(t *testing.T, name string, sc *scene.Scene, o *Options, scale float32) *image.RGBA {
	t.Helper()
	k := o.scale()
	w, h := int(sc.Width*k+0.5), int(sc.Height*k+0.5)
	bg := o.Look.ThemeOrDefault().Bg
	img := ui.Render(func(c *ui.Context) {
		ui.Column(c).Size(float32(w), float32(h)).Background(Color(bg, 1)).Children(func() {
			View(c, sc, o)
		})
	}, w, h, scale)
	path := filepath.Join("testdata", "golden", name+"-"+runtime.GOOS+".png")
	if *update {
		writePNG(t, path, img)
		return img
	}
	want, err := readPNG(path)
	if os.IsNotExist(err) {
		t.Skipf("no golden image %s for this system; run go test -update after checking the image", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	if diff := diffImages(want, img); diff != "" {
		out := filepath.Join(t.TempDir(), name+".png")
		writePNG(t, out, img)
		t.Errorf("%s differs from its golden image: %s; got %s", name, diff, out)
	}
	return img
}

func TestGoldenFlowchart(t *testing.T) {
	snap(t, "flowchart", samples.Flowchart(goFonts), &Options{Fonts: goFonts}, 1)
}

func TestGoldenFlowchartScaled(t *testing.T) {
	// Twice the size, on a screen of twice the density.
	snap(t, "flowchart-x2", samples.Flowchart(goFonts), &Options{Fonts: goFonts, Scale: 2}, 2)
}

func TestGoldenFeatures(t *testing.T) {
	snap(t, "features", samples.Features(goFonts), &Options{Fonts: goFonts, Draw: func(p *Painter, it *scene.Item, s style.Resolved, next func()) {
		if it.ID != "mine" {
			next()
			return
		}
		// A diamond of one's own: an ellipse in the accent color, with the label as it is.
		x, y := p.Point(it.Box.Center())
		ellipse := new(ui.Path).MoveTo(x+p.Len(it.Box.W/2), y)
		const k = 0.5522847498
		rx, ry := p.Len(it.Box.W/2), p.Len(it.Box.H/2)
		ellipse.CubeTo(x+rx, y+ry*k, x+rx*k, y+ry, x, y+ry).CubeTo(x-rx*k, y+ry, x-rx, y+ry*k, x-rx, y).
			CubeTo(x-rx, y-ry*k, x-rx*k, y-ry, x, y-ry).CubeTo(x+rx*k, y-ry, x+rx, y-ry*k, x+rx, y).Close()
		p.FillPath(ellipse, Color(style.Default().Accent, 0.3))
		p.StrokePath(ellipse, 1, Color(style.Default().Accent, 1))
		p.Label(it, s)
	}}, 1)
}

func TestGoldenDark(t *testing.T) {
	th := style.Default()
	th.Bg, th.Fg, th.Line = style.Hex("#1e1e1e"), style.Hex("#e6e6e6"), style.Hex("#e6e6e6")
	th.Surface, th.SurfaceAlt, th.Border, th.Muted = style.Hex("#2d2d2d"), style.Hex("#383838"), style.Hex("#8a8a8a"), style.Hex("#a0a0a0")
	look := style.Look{Theme: th, Overrides: style.Overrides{"flowchart.node": {Dash: style.Dashes(4, 2)}}}
	snap(t, "flowchart-dark", samples.Flowchart(goFonts), &Options{Fonts: goFonts, Look: look}, 1)
}

func readPNG(path string) (*image.RGBA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := m.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			rgba.Set(x, y, m.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return rgba, nil
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// diffImages describes how got differs from want, or returns "".
func diffImages(want, got *image.RGBA) string {
	if want.Bounds().Size() != got.Bounds().Size() {
		return fmt.Sprintf("size %v, want %v", got.Bounds().Size(), want.Bounds().Size())
	}
	n, first := 0, image.Point{-1, -1}
	b := want.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if want.RGBAAt(x, y) != got.RGBAAt(x, y) {
				if n == 0 {
					first = image.Pt(x, y)
				}
				n++
			}
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d pixels differ, the first at %v: %v, want %v", n, first, got.RGBAAt(first.X, first.Y), want.RGBAAt(first.X, first.Y))
}
