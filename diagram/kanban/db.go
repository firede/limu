package kanban

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
)

// A Node is a column or a card, Pele's KanbanNode. Its fields past Label are "" where Pele's
// are undefined; Defined tells those apart from ones set to "", as metadata can set them.
type Node struct {
	ID       string
	Level    int // the indentation of its line, in UTF-16 code units
	Label    string
	Icon     string
	Assigned string
	Ticket   string
	Priority string
	// CSSClasses are the classes its author gave it, as written after `:::`.
	CSSClasses string
	defined    uint8
}

// The fields of a Node that may be undefined.
const (
	definedIcon uint8 = 1 << iota
	definedAssigned
	definedTicket
	definedPriority
	definedClasses
)

// A Section is a column and its cards, Pele's KanbanSection.
type Section struct {
	Node
	Items []*Node
}

// Model is a parsed kanban board, Pele's KanbanModel.
type Model struct {
	Title    string
	Sections []*Section
}

// db builds a Model as Pele's KanbanDb does.
type db struct {
	model        *Model
	last         *Node
	sectionLevel int
	misplaced    *Node
}

func (d *db) GetType(start, end string) int { return common.OutlineType(start, end) }

func kanbanError(message string, code diagram.Code) *diagram.Error {
	return common.NewError(message, code, "kanban")
}

// AddNode adds a column, or a card to the last column, as Pele's addNode does.
func (d *db) AddNode(level int, id, descr string, _ int, shapeData string, hasData bool) error {
	node := &Node{ID: id, Level: level, Label: descr}

	if hasData {
		src := "{\n" + shapeData + "\n}"
		if strings.Contains(shapeData, "\n") {
			src = shapeData + "\n"
		}
		parsed, err := yaml.Parse(src)
		if err != nil {
			return kanbanError(`Metadata of "`+node.ID+`" is not valid YAML. `+err.Error(), diagram.Syntax)
		}
		// Mermaid throws a TypeError on metadata that is empty or not a mapping. Here it is
		// ignored.
		if doc, ok := parsed.(*yaml.Map); ok {
			shape := ""
			if v := doc.Value("shape"); yaml.Truthy(v) {
				shape = yaml.String(v)
			}
			if shape != chars.ToLower(shape) || strings.Contains(shape, "_") {
				return kanbanError("No such shape: "+shape+". Shape names should be lowercase.", diagram.Semantic)
			}
			set := func(key string, field *string, bit uint8) {
				if v := doc.Value(key); yaml.Truthy(v) {
					*field = yaml.String(v)
					node.defined |= bit
				}
			}
			set("label", &node.Label, 0)
			set("icon", &node.Icon, definedIcon)
			set("assigned", &node.Assigned, definedAssigned)
			set("ticket", &node.Ticket, definedTicket)
			set("priority", &node.Priority, definedPriority)
		}
	}

	// A node indented less than the first column becomes a card, and Mermaid rejects whatever
	// follows it.
	if d.misplaced != nil {
		return kanbanError(`Items without section detected, found section ("`+d.misplaced.Label+`")`, diagram.Semantic)
	}
	sections := d.model.Sections
	if len(sections) == 0 {
		d.sectionLevel = level
	}
	if level == d.sectionLevel {
		section := &Section{Node: *node, Items: []*Node{}}
		d.model.Sections = append(sections, section)
		d.last = &section.Node
		return nil
	}
	d.last = node
	last := sections[len(sections)-1]
	last.Items = append(last.Items, node)
	if level < d.sectionLevel {
		d.misplaced = node
	}
	return nil
}

// DecorateNode gives the last node an icon or classes, as Pele's decorateNode does. Mermaid
// throws a TypeError when a decoration comes before any node. Here it is ignored.
func (d *db) DecorateNode(text string, class bool) {
	node := d.last
	if node == nil || text == "" {
		return
	}
	if class {
		node.CSSClasses = text
		node.defined |= definedClasses
	} else {
		node.Icon = text
		node.defined |= definedIcon
	}
}
