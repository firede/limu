package treeview

import "github.com/firede/limu/diagram"

// The layout options of a tree view's config, as Pele's render.ts reads them.

// treeSettings are Mermaid's layout options of a tree view.
type treeSettings struct {
	paddingX, paddingY, rowIndent, lineThickness float64
}

// setting is a number of the diagram's config if it is at least lo and at most hi, or
// fallback, as Pele's setting reads it.
func setting(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["treeView"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo && v <= hi {
		return v
	}
	return fallback
}

// settings are the layout options of the config, with Pele's defaults, as Pele's
// treeSettings reads them.
func settings(config diagram.Config) treeSettings {
	return treeSettings{
		paddingX:      setting(config, "paddingX", 5, 0, 100),
		paddingY:      setting(config, "paddingY", 2, 0, 100),
		rowIndent:     setting(config, "rowIndent", 15, 0, 400),
		lineThickness: setting(config, "lineThickness", 1, 0.25, 12),
	}
}
