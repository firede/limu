package info

import (
	"math"
	"runtime/debug"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The roles of an info diagram's items: "info.title" for its title, and "info.version" for
// the renderer's name and version.
const (
	RoleTitle   scene.Role = "info.title"
	RoleVersion scene.Role = "info.version"
)

// module is limu's module path, by which a build records its version.
const module = "github.com/firede/limu"

// Version is limu's version as the program's build records it: the version of limu's module
// the program was built with ("v1.2.0"), or "(devel)" where the build records none, as when
// limu is built from a tree of its own sources.
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		m := &info.Main
		if m.Path != module {
			m = nil
			for _, d := range info.Deps {
				if d.Path == module {
					m = d
					break
				}
			}
		}
		if m != nil && m.Replace != nil {
			m = m.Replace
		}
		if m != nil && m.Version != "" {
			return m.Version
		}
	}
	return "(devel)"
}

// Layout lays an info diagram out as a scene, as Pele's renderInfo draws it: the title, if
// there is one, and under it the renderer's name and version. Pele draws its own, "Pele
// v1.7.0"; limu draws "limu" and its Version in its place.
func Layout(m *Model, _ diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return lay(m, o, "limu "+Version())
}

// lay is Pele's renderInfo, drawing name as the renderer's name and version.
func lay(m *Model, o *layout.Options, name string) *scene.Scene {
	size := o.Size()
	measure := o.Measure()
	pad := o.Pad()

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 8
	}
	version := text.Layout(name, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Icons: o.Icons})

	width := math.Ceil(max(title.Width, version.Width) + 2*pad)
	sc := &scene.Scene{
		Type:     "info",
		Width:    width,
		Height:   math.Ceil(titleHeight + version.Height + 2*pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	if !version.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleVersion, Label: &version,
			At:    scene.Point{X: width / 2, Y: pad + titleHeight + version.Height/2},
			Style: style.Style{Text: style.Muted},
		})
	}
	return sc
}
