package quadrant

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// db is what the parser fills, Pele's QuadrantDb.
type db struct {
	Model
}

func newDB() *db {
	return &db{Model{Classes: map[string]PointStyle{}}}
}

// invalid is the error of a style's value that cannot be read, as Pele's invalid words it.
func invalid(style, value, kind string) *diagram.Error {
	return common.NewError("value for "+style+" "+value+" is invalid, please use a valid "+kind, diagram.Semantic, "quadrantChart")
}

// keyValue is the first two parts of Mermaid's `style.trim().split(/\s*:\s*/)`, as Pele's
// keyValue finds them: the value is "undefined" when there is no colon.
func keyValue(style string) (string, string) {
	s := chars.Trim(style)
	colon := strings.IndexByte(s, ':')
	if colon == -1 {
		return s, "undefined"
	}
	rest := chars.TrimStart(s[colon+1:])
	next := strings.IndexByte(rest, ':')
	if next == -1 {
		return chars.TrimEnd(s[:colon]), rest
	}
	return chars.TrimEnd(s[:colon]), chars.TrimEnd(rest[:next])
}

// digits reports whether s is one or more ASCII digits and then suffix, as /^\d+$/ and
// /^\d+px$/ match.
func digits(s, suffix string) bool {
	body, ok := strings.CutSuffix(s, suffix)
	if !ok || body == "" {
		return false
	}
	for i := 0; i < len(body); i++ {
		if !chars.IsDigit(int(body[i])) {
			return false
		}
	}
	return true
}

// isHex reports whether s is a hex code of six or three digits, its # optional, as
// /^#?([\dA-Fa-f]{6}|[\dA-Fa-f]{3})$/ matches.
func isHex(s string) bool {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 && len(s) != 3 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !chars.IsHex(int(s[i])) {
			return false
		}
	}
	return true
}

// parseStyles reads a point's or a class's styles, as Pele's parseStyles does: a radius, a
// color, a stroke's color and its width, each checked, and later ones winning.
func parseStyles(styles []string) (PointStyle, error) {
	var out PointStyle
	for _, style := range styles {
		key, value := keyValue(style)
		switch key {
		case "radius":
			if !digits(value, "") {
				return out, invalid(key, value, "number")
			}
			r := chars.ParseInt(value, 10)
			out.Radius = &r
		case "color":
			if !isHex(value) {
				return out, invalid(key, value, "hex code")
			}
			out.Color = &value
		case "stroke-color":
			if !isHex(value) {
				return out, invalid(key, value, "hex code")
			}
			out.StrokeColor = &value
		case "stroke-width":
			if !digits(value, "px") {
				return out, invalid(key, value, "number of pixels (eg. 10px)")
			}
			out.StrokeWidth = &value
		default:
			return out, common.NewError("style named "+key+" is not supported.", diagram.Semantic, "quadrantChart")
		}
	}
	return out, nil
}

// trimmed is a label with the white space about its text taken off.
func trimmed(label Text) *Text {
	return &Text{Text: chars.Trim(label.Text), Markdown: label.Markdown}
}

// setQuadrantText sets the label of quadrant k+1, as Pele's setQuadrant1Text to
// setQuadrant4Text do.
func (d *db) setQuadrantText(k int, label Text) { d.Quadrants[k] = trimmed(label) }

func (d *db) setXAxisLeftText(label Text)   { d.XAxisLeft = trimmed(label) }
func (d *db) setXAxisRightText(label Text)  { d.XAxisRight = trimmed(label) }
func (d *db) setYAxisBottomText(label Text) { d.YAxisBottom = trimmed(label) }
func (d *db) setYAxisTopText(label Text)    { d.YAxisTop = trimmed(label) }

// addPoint adds a point, as Pele's addPoint does: its coordinates are read with Number.
func (d *db) addPoint(label Text, className, x, y string, styles []string) error {
	style, err := parseStyles(styles)
	if err != nil {
		return err
	}
	d.Points = append(d.Points, Point{PointStyle: style, X: chars.Number(x), Y: chars.Number(y), Text: *trimmed(label), Class: className})
	return nil
}

// addClass sets a class's styles. A class defined again keeps its place in the order.
func (d *db) addClass(className string, styles []string) error {
	style, err := parseStyles(styles)
	if err != nil {
		return err
	}
	if _, ok := d.Classes[className]; !ok {
		d.ClassOrder = append(d.ClassOrder, className)
	}
	d.Classes[className] = style
	return nil
}

func (d *db) setDiagramTitle(text string) { d.Title = text }

// setAccTitle drops the white space the title starts with, as /^\s+/ does.
func (d *db) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription drops the indentation of each line after the first.
func (d *db) setAccDescription(text string) { d.AccDescr = chars.DropIndents(text) }
