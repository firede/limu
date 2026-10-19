package treeview

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
	"github.com/firede/limu/scene"
)

// builtinPack is Mermaid's name for its pack of the tree view's own icons, Pele's
// BUILTIN_PACK. Only file and folder are in it; every other name is the host's to draw.
const builtinPack = "mermaid-treeview"

// glyphs are the outlines of the icons in the built-in pack, on a 24 by 24 grid, as Pele's
// GLYPHS has them.
var glyphs = map[string]string{
	"folder": "M3 6.5A1.5 1.5 0 0 1 4.5 5h4.6l2.4 2.5h8A1.5 1.5 0 0 1 21 9v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 18z",
	"file":   "M6.5 3h7L19 8.5v11a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 5 19.5v-15A1.5 1.5 0 0 1 6.5 3zM13.5 3v5.5H19",
}

// glyphPaths are the glyphs read into paths.
var glyphPaths = func() map[string]*scene.Path {
	out := map[string]*scene.Path{}
	for name, d := range glyphs {
		p, err := scene.ParsePath(d)
		if err != nil {
			panic(err)
		}
		out[name] = p
	}
	return out
}()

// glyphStroke is how wide Pele strokes a glyph, in units of its grid.
const glyphStroke = 1.5

// IconConfig is how a tree view shows icons, as its config sets it, Pele's IconConfig:
// whether entries show one without an icon() of their own, the pack a name without one is
// looked for in, and the icons that file names and extensions are shown with. A map is nil
// where the config has none.
type IconConfig struct {
	ShowIcons       bool
	DefaultIconPack string
	FilenameIcons   diagram.Config
	ExtensionIcons  diagram.Config
}

// own is the string a map holds under a key, as Pele's own reads it, and whether it holds
// one.
func own(m diagram.Config, key string) (string, bool) {
	s, ok := m[key].(string)
	return s, ok
}

// detectIcon is a file's icon from the maps of the config, as Pele's detectIcon finds it: a
// file name wins over an extension, and extensions match in lower case with or without the
// leading dot. It gives "" for none.
func detectIcon(name string, config IconConfig) string {
	if byName, _ := own(config.FilenameIcons, name); byName != "" {
		return byName
	}
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		ext := chars.ToLower(name[dot:])
		if icon, ok := own(config.ExtensionIcons, ext); ok {
			return icon
		}
		icon, _ := own(config.ExtensionIcons, ext[1:])
		return icon
	}
	return ""
}

// qualifyIcon is an icon's name with its pack, as Pele's qualifyIcon gives it.
func qualifyIcon(icon, defaultIconPack string) string {
	if strings.Contains(icon, ":") {
		return icon
	}
	if _, ok := glyphs[icon]; ok || defaultIconPack == "" {
		return builtinPack + ":" + icon
	}
	return defaultIconPack + ":" + icon
}

// NodeIcon is the qualified name of an entry's icon, or "" when it shows none, as Pele's
// getNodeIcon gives it. An icon() annotation always wins; otherwise icons appear only with
// ShowIcons.
func NodeIcon(node *Node, config IconConfig) string {
	if node.Icon == "none" {
		return ""
	}
	if node.Icon != "" {
		return qualifyIcon(node.Icon, config.DefaultIconPack)
	}
	if !config.ShowIcons {
		return ""
	}
	if node.Type == File {
		detected := detectIcon(node.Name, config)
		if detected == "none" {
			return ""
		}
		if detected != "" {
			return qualifyIcon(detected, config.DefaultIconPack)
		}
	}
	if node.Type == Directory {
		return builtinPack + ":folder"
	}
	return builtinPack + ":file"
}

// Icons is the icon settings of a diagram's config, as Pele's iconConfig reads them.
func Icons(config diagram.Config) IconConfig {
	c, _ := config["treeView"].(diagram.Config)
	maps := func(v any) diagram.Config {
		m, _ := v.(diagram.Config)
		return m
	}
	pack, _ := c["defaultIconPack"].(string)
	return IconConfig{
		ShowIcons:       yaml.Truthy(c["showIcons"]),
		DefaultIconPack: pack,
		FilenameIcons:   maps(c["filenameIcons"]),
		ExtensionIcons:  maps(c["extensionIcons"]),
	}
}
