package ishikawa

// A level is a node of the line of ancestors of the last node, with its level.
type level struct {
	level int
	node  *Node
}

// db builds a Model as Pele's IshikawaDb does.
type db struct {
	model *Model
	// stack is the line of ancestors of the last node, each with its level.
	stack     []level
	baseLevel int
	hasBase   bool
}

func newDB() *db { return &db{model: &Model{}} }

// addNode adds a line of the outline, as Pele's addNode does. The first line is the effect.
// Every later line is a cause, nested by its indentation relative to the first cause, so the
// effect itself may be indented more than the causes.
func (d *db) addNode(rawLevel int, text string) {
	node := &Node{Text: text}
	if d.model.Root == nil {
		d.model.Root = node
		d.stack = []level{{0, node}}
		return
	}
	if !d.hasBase {
		d.baseLevel, d.hasBase = rawLevel, true
	}
	lv := max(rawLevel-d.baseLevel+1, 1)
	for len(d.stack) > 1 && d.stack[len(d.stack)-1].level >= lv {
		d.stack = d.stack[:len(d.stack)-1]
	}
	parent := d.stack[len(d.stack)-1].node
	parent.Children = append(parent.Children, node)
	d.stack = append(d.stack, level{lv, node})
}
