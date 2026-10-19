package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
)

// sourceFilters are the files Open and Save offer: Mermaid's own, and Markdown and plain
// text, which may hold a diagram.
var sourceFilters = []mygo.FileFilter{
	{Name: "Mermaid", Extensions: []string{"mmd", "mermaid"}},
	{Name: "Markdown", Extensions: []string{"md"}},
	{Name: "Text", Extensions: []string{"txt"}},
}

// readSource reads a diagram's text from a file, without the byte order mark an editor may
// have written before it. From a Markdown file, it reads the first block fenced as
// mermaid, or the whole file when there is none.
func readSource(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sourceOf(path, string(b)), nil
}

// sourceOf is the diagram's text in the content of a file named path.
func sourceOf(path, content string) string {
	content = strings.TrimPrefix(content, "\uFEFF")
	if isMarkdown(path) {
		if block, ok := mermaidBlock(content); ok {
			return block
		}
	}
	return content
}

// writeSource writes a diagram's text to a file: in a Markdown file, as a block fenced as
// mermaid, which readSource reads back.
func writeSource(path, src string) error {
	return os.WriteFile(path, []byte(fileContent(path, src)), 0o644)
}

// fileContent is what a file named path holds for a diagram's text.
func fileContent(path, src string) string {
	if !strings.HasSuffix(src, "\n") {
		src += "\n"
	}
	if isMarkdown(path) {
		return "```mermaid\n" + src + "```\n"
	}
	return src
}

func isMarkdown(path string) bool { return strings.EqualFold(filepath.Ext(path), ".md") }

// mermaidBlock is the first block of Markdown fenced with ``` or ~~~ and the info string
// mermaid, and whether there is one. A block left open runs to the end of the text.
func mermaidBlock(md string) (string, bool) {
	lines := strings.SplitAfter(md, "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		fence := ""
		for _, f := range []string{"```", "~~~"} {
			if strings.HasPrefix(t, f) {
				fence = t[:len(t)-len(strings.TrimLeft(t, f[:1]))]
			}
		}
		if fence == "" || strings.TrimSpace(t[len(fence):]) != "mermaid" {
			continue
		}
		var b strings.Builder
		for _, body := range lines[i+1:] {
			bt := strings.TrimSpace(body)
			if strings.HasPrefix(bt, fence) && strings.Trim(bt, fence[:1]) == "" {
				break
			}
			b.WriteString(body)
		}
		return b.String(), true
	}
	return "", false
}
