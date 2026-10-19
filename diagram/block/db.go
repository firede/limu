package block

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// maxSpaces is the most space blocks a diagram may hold, Pele's MAX_SPACES.
const maxSpaces = 100000

// typeStr2Type is the type of a block whose label is in the brackets typeStr, "" for none, as
// Pele's typeStr2Type gives it.
func typeStr2Type(typeStr string) string {
	switch typeStr {
	case "[]":
		return "square"
	case "()":
		return "round"
	case "(())":
		return "circle"
	case ">]":
		return "rect_left_inv_arrow"
	case "{}":
		return "diamond"
	case "{{}}":
		return "hexagon"
	case "([])":
		return "stadium"
	case "[[]]":
		return "subroutine"
	case "[()]":
		return "cylinder"
	case "((()))":
		return "doublecircle"
	case "[//]":
		return "lean_right"
	case `[\\]`:
		return "lean_left"
	case `[/\]`:
		return "trapezoid"
	case `[\/]`:
		return "inv_trapezoid"
	case "<[]>":
		return "block_arrow"
	}
	return "na"
}

// lastUnit is the last code unit of s as JavaScript's slice(-1) takes it, as a string.
func lastUnit(s string) string {
	if s == "" {
		return ""
	}
	if c := s[len(s)-1]; c < 0x80 {
		return s[len(s)-1:]
	}
	return "�"
}

// edgeStrToEdgeData is the marker at the end of a link, as Pele's edgeStrToEdgeData reads it.
func edgeStrToEdgeData(typeStr string) string {
	switch lastUnit(chars.Trim(typeStr)) {
	case "x":
		return "arrow_cross"
	case "o":
		return "arrow_circle"
	case ">":
		return "arrow_point"
	}
	return ""
}

// edgeStrToEdgeStartData is the marker at the start of a link, as Pele's
// edgeStrToEdgeStartData reads it.
func edgeStrToEdgeStartData(typeStr string) string {
	s := chars.Trim(typeStr)
	if s != "" {
		switch s[0] {
		case 'x':
			return "arrow_cross"
		case 'o':
			return "arrow_circle"
		case '<':
			return "arrow_point"
		}
	}
	return "arrow_open"
}

// edgeStrToThickness is how thick a link is drawn, as Pele's edgeStrToThickness says.
func edgeStrToThickness(typeStr string) string {
	if strings.Contains(typeStr, "==") {
		return "thick"
	}
	return "normal"
}

// edgeStrToPattern is how a link's line is drawn, as Pele's edgeStrToPattern says.
func edgeStrToPattern(typeStr string) string {
	if strings.Contains(typeStr, ".-") {
		return "dotted"
	}
	return "solid"
}

// A hierarchyFrame is a statement list being read into a block's children, Pele's Frame.
type hierarchyFrame struct {
	list     []*Block
	at       int
	parent   *Block
	children []*Block
	columns  float64
}

// newFrame is the frame of the statements of parent, Pele's frame: the statements flattened,
// and the columns the first column setting among them gives.
func newFrame(statements []statement, parent *Block) *hierarchyFrame {
	var list []*Block
	for _, s := range statements {
		list = append(list, s.blocks...)
	}
	columns := -1.0
	for _, b := range list {
		if b.Type == "column-setting" {
			if b.Columns != nil {
				columns = *b.Columns
			}
			break
		}
	}
	return &hierarchyFrame{list: list, parent: parent, children: []*Block{}, columns: columns}
}

// DB is a parsed block diagram, Pele's BlockDb: the tree of blocks under Root, the edges,
// the classes classDef statements define, and every block by its id.
type DB struct {
	Title  string
	Root   *Block
	Blocks []*Block
	Edges  []*Block
	// Classes are the classes by name, and ClassOrder their names in the order they were
	// first defined.
	Classes    map[string]*ClassDef
	ClassOrder []string
	// ByID is every block by its id, and IDOrder the ids in the order they were first set.
	// Mermaid keeps the root in here too, which makes a block called `root` vanish; here it
	// stays out.
	ByID    map[string]*Block
	IDOrder []string
	// Warn, when set, is told of a block wider than the columns of its composite.
	Warn func(message string)

	// Ids that a class or style statement named before any block had them.
	pending   map[string]bool
	edgeCount map[string]int
	ids       int
	colors    int
	spaces    float64
}

// NewDB is a model with nothing in it, as Pele's BlockDb starts.
func NewDB() *DB {
	columns := -1.0
	return &DB{
		Root:      &Block{ID: "root", Type: "composite", Children: []*Block{}, Columns: &columns},
		Blocks:    []*Block{},
		Classes:   map[string]*ClassDef{},
		ByID:      map[string]*Block{},
		pending:   map[string]bool{},
		edgeCount: map[string]int{},
	}
}

// generateID is the next id of a block that has none of its own, as Pele's generateId gives
// it. Mermaid's ids are random; these are counted, so the same text always gives the same
// output.
func (db *DB) generateID() string {
	db.ids++
	return "id-" + strconv.Itoa(db.ids)
}

// setByID files a block under its id.
func (db *DB) setByID(id string, b *Block) {
	if _, ok := db.ByID[id]; !ok {
		db.IDOrder = append(db.IDOrder, id)
	}
	db.ByID[id] = b
}

// addStyleClass adds the declarations of a classDef statement to a class, as Pele's
// addStyleClass does.
func (db *DB) addStyleClass(id, styleAttributes string) {
	found, ok := db.Classes[id]
	if !ok {
		found = &ClassDef{ID: id, Styles: []string{}, TextStyles: []string{}}
		db.Classes[id] = found
		db.ClassOrder = append(db.ClassOrder, id)
	}
	for _, attribute := range strings.Split(styleAttributes, ",") {
		fixed := attribute
		if semi := strings.IndexByte(attribute, ';'); semi != -1 {
			fixed = attribute[:semi] + attribute[semi+1:]
		}
		fixed = chars.Trim(fixed)
		if strings.Contains(attribute, "color") {
			found.TextStyles = append(found.TextStyles, strings.Replace(strings.Replace(fixed, "fill", "bgFill", 1), "color", "fill", 1))
		}
		found.Styles = append(found.Styles, fixed)
	}
}

// addStyle2Node gives a block the declarations of a style statement, as Pele's
// addStyle2Node does.
func (db *DB) addStyle2Node(id, styles string) {
	db.named(id).Styles = strings.Split(styles, ",")
}

// setCssClass gives blocks a class, as Pele's setCssClass does. Mermaid looks each id up
// before trimming it, so `class a, b hot` misses b; here it is trimmed first.
func (db *DB) setCssClass(itemIDs, cssClassName string) {
	for _, id := range strings.Split(itemIDs, ",") {
		b := db.named(chars.Trim(id))
		if b.Classes == nil {
			b.Classes = []string{}
		}
		b.Classes = append(b.Classes, cssClassName)
	}
}

// named is the block of an id, or one that waits for it, as Pele's named gives it. Mermaid
// throws on a style for an unknown id, and a class for one makes the block of that id
// disappear when it is declared later; here both wait for the block.
func (db *DB) named(id string) *Block {
	b, ok := db.ByID[id]
	if !ok {
		b = &Block{ID: id, Type: "na", Children: []*Block{}}
		db.setByID(id, b)
		db.pending[id] = true
	}
	return b
}

// setHierarchy reads the statements of a diagram into its tree of blocks, its edges and its
// classes, as Pele's setHierarchy does.
func (db *DB) setHierarchy(statements []statement) error {
	stack := []*hierarchyFrame{newFrame(statements, db.Root)}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		if f.at == len(f.list) {
			f.parent.Children = f.children
			stack = stack[:len(stack)-1]
			continue
		}
		b := f.list[f.at]
		f.at++
		if f.columns > 0 && b.Type != "column-setting" && b.WidthInColumns != nil && *b.WidthInColumns > f.columns && db.Warn != nil {
			db.Warn("Block " + b.ID + " width " + chars.NumberString(*b.WidthInColumns) +
				" exceeds configured column width " + chars.NumberString(f.columns))
		}
		switch b.Type {
		case "classDef":
			db.addStyleClass(b.ID, b.CSS)
			continue
		case "applyClass":
			db.setCssClass(b.ID, b.StyleClass)
			continue
		case "applyStyles":
			if b.StylesStr != "" {
				db.addStyle2Node(b.ID, b.StylesStr)
			}
			continue
		case "column-setting":
			columns := -1.0
			if b.Columns != nil {
				columns = *b.Columns
			}
			f.parent.Columns = &columns
			continue
		case "edge":
			count := db.edgeCount[b.ID] + 1
			db.edgeCount[b.ID] = count
			b.ID = strconv.Itoa(count) + "-" + b.ID
			db.Edges = append(db.Edges, b)
			continue
		}

		// A composite whose header was a link statement has no id of its own.
		if b.ID == "" {
			b.ID = db.generateID()
		}
		if !b.HasLabel || b.Label == "" {
			if b.Type == "composite" {
				b.Label = ""
			} else {
				b.Label = b.ID
			}
			b.HasLabel = true
		}
		known, ok := db.ByID[b.ID]
		fresh := !ok || db.pending[b.ID]
		delete(db.pending, b.ID)
		if !fresh {
			if b.Type != "na" {
				known.Type = b.Type
			}
			if b.Label != b.ID {
				known.Label = b.Label
				known.HasLabel = true
			}
		} else {
			if ok {
				b.Classes = known.Classes
				b.Styles = known.Styles
			}
			if b.Type == "composite" {
				index := db.colors
				db.colors++
				b.ColorIndex = &index
			}
			db.setByID(b.ID, b)
		}

		inner := b.statements
		b.statements = nil
		b.Children = []*Block{}
		if b.Type == "space" {
			width := 1.0
			if b.Width != nil {
				width = *b.Width
			}
			if !(width <= maxSpaces-db.spaces) {
				return common.NewError("A block diagram can hold at most "+strconv.Itoa(maxSpaces)+" space blocks.", diagram.Limit, "block")
			}
			db.spaces += max(width, 0)
			for j := 0; float64(j) < width; j++ {
				copied := *b
				copied.ID = b.ID + "-" + strconv.Itoa(j)
				copied.Children = []*Block{}
				db.setByID(copied.ID, &copied)
				f.children = append(f.children, &copied)
			}
		} else if fresh {
			f.children = append(f.children, b)
		}
		if len(inner) > 0 {
			stack = append(stack, newFrame(inner, b))
		}
	}
	db.Blocks = db.Root.Children
	if db.Blocks == nil {
		db.Blocks = []*Block{}
	}
	return nil
}
