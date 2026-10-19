package block

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// nodeInfo is a block as a statement names it, Pele's NodeInfo.
type nodeInfo struct {
	id       string
	label    string
	hasLabel bool
	// typeStr is the brackets about the label, "" for none.
	typeStr    string
	directions []string
}

// A statement is a block, or a list of them: a statement with links is a block, an edge, a
// block and so on, Pele's Statement.
type statement struct {
	blocks []*Block
	list   bool
}

// one is a statement of a single block.
func one(b *Block) statement { return statement{blocks: []*Block{b}} }

// frame is a composite being read: the statement list of its parent, and its header,
// Pele's Frame.
type parseFrame struct {
	list      []statement
	header    statement
	hasHeader bool
}

const statementStart = "'NODE_ID', 'COLUMNS', 'SPACE_BLOCK', 'BLOCK_DIAGRAM_KEY', 'id-block', 'classDef', 'class', 'style'"

// closeDocument inlines a document's last statement when it is a list, as the grammar does,
// Pele's closeDocument.
func closeDocument(list []statement) []statement {
	if len(list) == 0 || !list[len(list)-1].list {
		return list
	}
	last := list[len(list)-1]
	list = list[:len(list)-1]
	for _, b := range last.blocks {
		list = append(list, one(b))
	}
	return list
}

// parseFailure carries a syntax error out of the parser.
type parseFailure struct{ err *diagram.Error }

// parseBlock parses a block diagram's text into db, as Pele's parseBlock does.
func parseBlock(src string, db *DB) (err error) {
	tk := tokenize(src)
	types, texts := tk.Types, tk.Texts
	i := 0
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(parseFailure)
			if !ok {
				panic(r)
			}
			err = f.err
		}
	}()

	fail := func(expected string) {
		t := types[i]
		if t == tERROR {
			panic(parseFailure{diagram.SyntaxError("block", src, tk.Starts[i], "", true)})
		}
		panic(parseFailure{diagram.SyntaxError("block", src, tk.Starts[i], "Expecting "+expected+", got '"+tokenNames[t]+"'", false)})
	}
	expect := func(typ int) string {
		if types[i] != typ {
			fail("'" + tokenNames[typ] + "'")
		}
		i++
		return texts[i-1]
	}

	node := func() nodeInfo {
		id := expect(tNODE_ID)
		if types[i] == tNODE_DSTART {
			open := texts[i]
			i++
			label := expect(tSTR)
			return nodeInfo{id: id, label: label, hasLabel: true, typeStr: open + expect(tNODE_DEND)}
		}
		if types[i] == tBLOCK_ARROW_START {
			open := texts[i]
			i++
			label := expect(tSTR)
			dirs := []string{expect(tDIR)}
			for types[i] == tDIR {
				dirs = append(dirs, texts[i])
				i++
			}
			return nodeInfo{id: id, label: label, hasLabel: true, typeStr: open + expect(tBLOCK_ARROW_END), directions: dirs}
		}
		return nodeInfo{id: id}
	}
	typeOf := func(n nodeInfo) string { return typeStr2Type(n.typeStr) }

	nodeStatement := func() statement {
		first := node()
		size := 1.0
		if types[i] == tSIZE {
			size = chars.ParseInt(texts[i], 10)
			i++
		}
		from := &Block{ID: first.id, Label: first.label, HasLabel: first.hasLabel, Type: typeOf(first), Directions: first.directions, WidthInColumns: &size}
		var chain []*Block
		for types[i] == tLINK || types[i] == tSTART_LINK {
			label := ""
			t := types[i]
			i++
			if t == tSTART_LINK {
				expect(tLINK_LABEL)
				label = expect(tSTR)
				expect(tLINK)
			}
			link := texts[i-1]
			to := node()
			arrowTypeEnd := edgeStrToEdgeData(link)
			arrowTypeStart := edgeStrToEdgeStartData(link)
			thickness := edgeStrToThickness(link)
			pattern := edgeStrToPattern(link)
			// Mermaid keeps only the last link of `A --> B --> C` and loses its start. Here
			// every link counts.
			if chain == nil {
				chain = []*Block{{ID: from.ID, Label: from.Label, HasLabel: from.HasLabel, Type: from.Type, Directions: from.Directions}}
			}
			from = &Block{ID: to.id, Label: to.label, HasLabel: to.hasLabel, Type: typeOf(to), Directions: to.directions}
			last := chain[len(chain)-1]
			chain = append(chain, &Block{
				ID:             last.ID + "-" + to.id,
				Start:          last.ID,
				End:            to.id,
				Label:          label,
				HasLabel:       true,
				Type:           "edge",
				Thickness:      thickness,
				Pattern:        pattern,
				Directions:     to.directions,
				ArrowTypeEnd:   arrowTypeEnd,
				ArrowTypeStart: arrowTypeStart,
			}, from)
		}
		if chain != nil {
			return statement{blocks: chain, list: true}
		}
		return one(from)
	}

	expect(tKEY)
	// Composites nest without recursion: each open one keeps the statement list of its
	// parent.
	var open []parseFrame
	var list []statement
	for {
		switch types[i] {
		case tNODE_ID:
			list = append(list, nodeStatement())
		case tCOLUMNS:
			columns := chars.ParseInt(texts[i], 10)
			i++
			list = append(list, one(&Block{Type: "column-setting", Columns: &columns}))
		case tSPACE_BLOCK:
			width := chars.ParseInt(texts[i], 10)
			i++
			list = append(list, one(&Block{ID: db.generateID(), Type: "space", HasLabel: true, Width: &width, Children: []*Block{}}))
		case tKEY:
			i++
			open = append(open, parseFrame{list: list})
			list = nil
		case tID_BLOCK:
			i++
			if types[i] != tNODE_ID {
				fail("'NODE_ID'")
			}
			open = append(open, parseFrame{list: list, header: nodeStatement(), hasHeader: true})
			list = nil
		case tEnd:
			if len(open) == 0 || len(list) == 0 {
				fail(statementStart)
			}
			f := open[len(open)-1]
			open = open[:len(open)-1]
			i++
			id := db.generateID()
			children := closeDocument(list)
			list = f.list
			var b *Block
			switch {
			case !f.hasHeader:
				b = &Block{ID: id, Type: "composite", HasLabel: true}
			case f.header.list:
				// A header with links spreads as a list would, into fields no one reads: the
				// composite has no id and no label of its own.
				b = &Block{Type: "composite"}
			default:
				copied := *f.header.blocks[0]
				b = &copied
				b.Type = "composite"
			}
			b.statements = children
			list = append(list, one(b))
		case tCLASSDEF:
			i++
			id := chars.Trim(expect(tCLASSDEF_ID))
			list = append(list, one(&Block{Type: "classDef", ID: id, CSS: chars.Trim(expect(tCLASSDEF_STYLEOPTS))}))
		case tCLASS:
			i++
			id := chars.Trim(expect(tCLASSENTITY_IDS))
			list = append(list, one(&Block{Type: "applyClass", ID: id, StyleClass: chars.Trim(expect(tSTYLECLASS))}))
		case tSTYLE:
			i++
			id := chars.Trim(expect(tSTYLE_ENTITY_IDS))
			list = append(list, one(&Block{Type: "applyStyles", ID: id, StylesStr: chars.Trim(expect(tSTYLE_DEFINITION_DATA))}))
		case tEOF:
			if len(open) > 0 {
				fail(statementStart + ", 'end'")
			}
			if len(list) == 0 {
				fail(statementStart)
			}
			return db.setHierarchy(closeDocument(list))
		default:
			switch {
			case len(list) == 0:
				fail(statementStart)
			case len(open) > 0:
				fail(statementStart + ", 'end'")
			default:
				fail(statementStart + ", 'EOF'")
			}
		}
	}
}
