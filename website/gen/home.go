package main

import "log"

// The home page's picture of where the pipeline branches, drawn by limu: a renderer of SVG
// goes from text to a layout to SVG, for a browser; limu goes from text to a layout to a
// scene, and branches there, to a backend. The page marks the scene in its accent.
const (
	rendererSource = `flowchart LR
  text[Mermaid text] --> layout[Layout] --> svg[SVG] --> browser[Browser]`
	limuSource = `flowchart LR
  text[Mermaid text] --> layout[Layout] --> scene[Scene]
  scene --> mygo[MyGo, on the GPU]
  scene --> svg[SVG]
  scene --> own[A backend of your own]`
)

type homeDrawings struct {
	Renderer drawing `json:"renderer"`
	Limu     drawing `json:"limu"`
}

func home() homeDrawings {
	r, err := draw(rendererSource)
	if err != nil {
		log.Fatal(err)
	}
	l, err := draw(limuSource)
	if err != nil {
		log.Fatal(err)
	}
	return homeDrawings{r, l}
}
