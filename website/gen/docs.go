package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// docsDiagrams draws the diagrams the documents show: each Mermaid code block marked draw,
// as
//
//	```mermaid draw title="limu"
//
// keyed by its source as Markdown reads it, its lines joined by line feeds, so that the page
// puts the drawing in the code block's place. The page's Markdown, which agents read, keeps
// the source.
func docsDiagrams() map[string]drawing {
	files, err := filepath.Glob("src/content/docs/*.md")
	if err != nil {
		log.Fatal(err)
	}
	out := map[string]drawing{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		for _, src := range drawnBlocks(string(b)) {
			d, err := draw(src)
			if err != nil {
				log.Fatalf("gen: %s: %v\n%s", f, err, src)
			}
			out[src] = d
		}
	}
	return out
}

// drawnBlocks are the sources of the Mermaid code blocks in Markdown whose info string marks
// them draw.
func drawnBlocks(md string) []string {
	var blocks []string
	var block []string
	in := false
	for line := range strings.Lines(md) {
		line = strings.TrimSuffix(line, "\n")
		switch {
		case !in && strings.HasPrefix(line, "```"):
			fields := strings.Fields(strings.TrimPrefix(line, "```"))
			in = true
			block = nil
			if len(fields) > 1 && fields[0] == "mermaid" && fields[1] == "draw" {
				block = []string{}
			}
		case in && strings.TrimSpace(line) == "```":
			in = false
			if block != nil {
				blocks = append(blocks, strings.Join(block, "\n"))
			}
		case in && block != nil:
			block = append(block, line)
		}
	}
	return blocks
}
