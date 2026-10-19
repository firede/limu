package limu

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/flowchart"
	"github.com/firede/limu/diagram/usecase"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// Layout reads a diagram's text and lays it out as a scene, for a backend to draw. Its
// errors are *diagram.Error: a diagram of a type limu does not read yet, text it cannot
// parse, or text past a limit.
func Layout(src string, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	p, err := diagram.Prepare(src, diagram.Options{})
	if err != nil {
		return nil, err
	}
	var sc *scene.Scene
	switch p.Type {
	case diagram.Flowchart:
		db, err := flowchart.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = flowchart.Layout(db, p.Config, o)
	case diagram.UseCase:
		m, err := usecase.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = usecase.Layout(m, p.Config, o)
	default:
		return nil, diagram.Unsupported(p.Type)
	}
	if o.Snap > 0 {
		sc.Snap(o.Snap)
	}
	return sc, nil
}
