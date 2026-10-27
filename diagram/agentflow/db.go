package agentflow

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/internal/yaml"
)

// LabelType is how a label's text is read: "text", "string" or "markdown".
type LabelType string

// The types of label.
const (
	LabelText     LabelType = "text"
	LabelString   LabelType = "string"
	LabelMarkdown LabelType = "markdown"
)

// flowText is a label as written, with how it is read. Pele's FlowText.
type flowText struct {
	Text string
	Type LabelType
}

// loc is a span of the source as Jison reports it: lines from 1, columns from 0, in UTF-16
// code units. Pele's Loc.
type loc struct {
	FirstLine, FirstColumn, LastLine, LastColumn int
}

// Prop is a property a node is given with `id[|name:value|label]`.
type Prop struct {
	Key, Value string
}

// Node is a node of an agent flow, or a connector, Pele's AgentNode.
type Node struct {
	ID        string
	Text      string
	LabelType LabelType
	// Type is the shape as the text gives it: a bracket's name, as "square" or "diamond", or
	// the name of a shape the metadata gives, and "" where it gives none.
	Type    string
	Styles  []string
	Classes []string
	Dir     string
	// Props are the node's properties, in the order JavaScript gives an object's keys.
	Props      []Prop
	Link       string
	LinkTarget string
	// Metadata is what metadata blocks give the node, nil where none do.
	Metadata    *yaml.Map
	IsConnector bool
}

// Edge is an edge of an agent flow, Pele's AgentEdge.
type Edge struct {
	Start     string
	End       string
	Type      string
	Text      string
	LabelType LabelType
	Stroke    string
	// EdgeSemantic is what the arrow means: "sequence", "reference", "failure", or "".
	EdgeSemantic string
	// Length is the edge's length in ranks, at most 10, and 0 where it has none, as an edge
	// of a start and an end that do not agree.
	Length int
	// Style is what linkStyle sets, nil where it sets nothing.
	Style           []string
	Classes         []string
	ID              string
	IsUserDefinedID bool
	// Interpolate is the edge's curve: a string from linkStyle, or whatever value the edge's
	// metadata gives its curve. Animate and Animation are what its metadata gives them. Each
	// is nil where unset or null, which no reader tells apart.
	Interpolate any
	Animate     any
	Animation   any
	Metadata    *yaml.Map
}

// ClassDef is a class of styles, from classDef. Pele's AgentClassDef.
type ClassDef struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// Subgraph is a flow, Pele's AgentSubgraph.
type Subgraph struct {
	ID        string
	Nodes     []string
	Title     string
	Classes   []string
	Dir       string
	LabelType LabelType
	Type      string // always "flow"
	Metadata  *yaml.Map
}

// Tooltip is a node's tooltip, from click.
type Tooltip struct {
	ID, Text string
}

// linkInfo is what an arrow says of an edge: Pele's LinkInfo.
type linkInfo struct {
	Type         string
	Stroke       string
	Length       int
	EdgeSemantic string
	Text         *flowText
	ID           string
	HasID        bool
}

// ElementPosition is where a statement is in the source: lines from 1, counting the front
// matter, columns from 0. The indexes are always 0, as Mermaid's are.
type ElementPosition struct {
	StartLine, StartColumn, EndLine, EndColumn, StartIndex, EndIndex int
}

// The kinds of statement a mapping records.
const (
	StatementVertex     = "vertex"
	StatementEdge       = "edge"
	StatementSubgraph   = "subgraph"
	StatementConnector  = "connector"
	StatementAttachment = "attachment"
)

// ElementMapping says where a statement is, Pele's ElementMapping.
type ElementMapping struct {
	ID       string
	Type     string
	Position *ElementPosition
}

// The diagnostics a parse records, by Mermaid's ids.
const (
	ShapeUnsupported     = "SHAPE_UNSUPPORTED"
	ShapeRemoved         = "SHAPE_REMOVED"
	ContainmentViolation = "CONTAINMENT_VIOLATION"
)

// Diagnostic is a finding about the meaning of a diagram that does not stop it being drawn,
// Pele's Diagnostic.
type Diagnostic struct {
	ID       string
	Severity string // "warning" or "error"
	Message  string
	NodeID   string
	EdgeID   string
	// Position is the place of the first statement of the node or edge, nil where it has none.
	Position *ElementPosition
}

// options are Pele's AgentflowDbOptions.
type options struct {
	// MaxEdges is the most edges the flow may have; no limit when it is negative.
	MaxEdges int
	// InheritDir gives a flow that sets no direction the diagram's.
	InheritDir bool
	// Direction is the direction InheritDir gives where the diagram has none.
	Direction string
	// LineOffset is the number of lines of front matter before the text the parser sees,
	// added to every line reported where Source is not set.
	LineOffset int
	// Source is the text the parser reads, where errors in metadata are located.
	Source string
	// Log, if set, is told of each diagnostic as it is recorded.
	Log func(severity, message string)
}

// DB is what an agent flow's text says, as Pele's AgentflowDb holds it: its nodes, edges,
// flows, connectors and the rest, in the order they were first seen.
type DB struct {
	Direction string
	// Nodes are the nodes, connectors among them, in the order they were first seen; Node
	// finds one by its id.
	Nodes     []*Node
	Edges     []*Edge
	Subgraphs []*Subgraph
	// Connectors are the connectors in the order they were declared.
	Connectors []*Node
	// Classes are the classes in the order they were first defined; Class finds one.
	Classes []*ClassDef
	// Tooltips are the tooltips in the order their nodes were first given one.
	Tooltips []Tooltip
	// DefaultEdgeStyle is what `linkStyle default` sets, nil where it is not set.
	DefaultEdgeStyle []string
	// DefaultInterpolate is what `linkStyle default interpolate` sets, "" where it is not set.
	DefaultInterpolate string
	// Mappings say where each statement is in the source, in the order they were read.
	Mappings    []*ElementMapping
	Diagnostics []*Diagnostic
	Title       string
	AccTitle    string
	AccDescr    string
	LineOffset  int

	nodes          map[string]int
	connectors     map[string]*Node
	classes        map[string]*ClassDef
	tooltips       map[string]int
	subgraphLookup map[string]*Subgraph
	globalNodes    map[string]bool
	members        map[string]bool
	edgeByID       map[string]*Edge
	pairCount      map[string]int
	subCount       int
	bareMappings   map[*ElementMapping]bool
	firstMapping   map[string]*ElementMapping
	reported       map[string][]*Diagnostic
	options        options
	graph          *graph
}

// newDB is an empty agent flow.
// It is the constructor of Pele's AgentflowDb.
func newDB(options options) *DB {
	return &DB{
		LineOffset:     options.LineOffset,
		nodes:          map[string]int{},
		connectors:     map[string]*Node{},
		classes:        map[string]*ClassDef{},
		tooltips:       map[string]int{},
		subgraphLookup: map[string]*Subgraph{},
		globalNodes:    map[string]bool{},
		members:        map[string]bool{},
		edgeByID:       map[string]*Edge{},
		pairCount:      map[string]int{},
		bareMappings:   map[*ElementMapping]bool{},
		firstMapping:   map[string]*ElementMapping{},
		reported:       map[string][]*Diagnostic{},
		options:        options,
	}
}

// Node is the node of id, or nil.
// It is Pele's nodes.get.
func (d *DB) Node(id string) *Node {
	if i, ok := d.nodes[id]; ok {
		return d.Nodes[i]
	}
	return nil
}

// setNode sets the node of id, in the place of the one it replaces, as Map.set does.
// It is Pele's nodes.set.
func (d *DB) setNode(n *Node) {
	if i, ok := d.nodes[n.ID]; ok {
		d.Nodes[i] = n
		return
	}
	d.nodes[n.ID] = len(d.Nodes)
	d.Nodes = append(d.Nodes, n)
}

// Connector is the connector of id, or nil.
// It is Pele's connectors.get.
func (d *DB) Connector(id string) *Node { return d.connectors[id] }

// Class is the class of id, or nil.
// It is Pele's classes.get.
func (d *DB) Class(id string) *ClassDef { return d.classes[id] }

// Tooltip is the tooltip of the node of id, and whether it has one.
// It is Pele's tooltips.get.
func (d *DB) Tooltip(id string) (string, bool) {
	i, ok := d.tooltips[id]
	if !ok {
		return "", false
	}
	return d.Tooltips[i].Text, true
}

// Subgraph is the flow of id, or nil.
// It is Pele's subgraph.
func (d *DB) Subgraph(id string) *Subgraph { return d.subgraphLookup[id] }

// presentationKeys are the keys of metadata that say only how a thing is drawn, left out of
// the semantic model.
var presentationKeys = map[string]bool{
	"shape": true, "view": true, "icon": true, "img": true, "form": true, "pos": true, "w": true, "h": true,
	"class": true, "style": true, "labelType": true,
}

// toolShapes are the shapes a tool is drawn with, Pele's TOOL_SHAPES.
var toolShapes = map[string]bool{"subroutine": true, "subprocess": true, "subproc": true, "framed-rectangle": true, "tool": true}

// isToolDefinition reports whether a node is drawn as a tool, as Pele's isToolDefinition does.
func isToolDefinition(n *Node) bool { return toolShapes[n.Type] }

// labelType is the label type a metadata block's labelType names, Markdown for any other, as
// Pele's labelType gives it.
func labelType(t string) LabelType {
	switch t {
	case "markdown", "string", "text":
		return LabelType(t)
	}
	return LabelMarkdown
}

// unquote is text without the double quotes it is written between, if it is, as Pele's
// unquote gives it.
func unquote(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

// stripPrototypeKeys copies a value read from YAML without its __proto__, constructor and
// prototype keys, at every level, so that a reader merging metadata into its own objects
// cannot be made to pollute a prototype. Pele's stripPrototypeKeys.
func stripPrototypeKeys(v yaml.Value) yaml.Value {
	switch x := v.(type) {
	case []yaml.Value:
		out := make([]yaml.Value, len(x))
		for i, item := range x {
			out[i] = stripPrototypeKeys(item)
		}
		return out
	case *yaml.Map:
		out := yaml.NewMap()
		for k, item := range x.All() {
			if k == "__proto__" || k == "constructor" || k == "prototype" {
				continue
			}
			out.Set(k, stripPrototypeKeys(item))
		}
		return out
	}
	return v
}

// merge adds what a metadata block holds to an element's metadata, as Pele's merge does with
// Object.assign: Mermaid spreads the parsed YAML whatever it is, so a list adds its indexes as
// keys and a string its code units. A code unit that is half of a surrogate pair, which a Go
// string cannot hold alone, is U+FFFD, as JSON reads one back.
func merge(into *yaml.Map, doc yaml.Value) *yaml.Map {
	if into == nil {
		into = yaml.NewMap()
	}
	switch x := doc.(type) {
	case *yaml.Map:
		for k, v := range x.All() {
			into.Set(k, v)
		}
	case []yaml.Value:
		for i, v := range x {
			into.Set(strconv.Itoa(i), v)
		}
	case string:
		i := 0
		for _, r := range x {
			if r > 0xffff {
				into.Set(strconv.Itoa(i), "�")
				into.Set(strconv.Itoa(i+1), "�")
				i += 2
				continue
			}
			into.Set(strconv.Itoa(i), string(r))
			i++
		}
	}
	return into
}

// blockScalar reports whether code ends a line with the header of a block scalar:
// /:\s*[>|][\d+-]*\s*$/.
// It is Pele's RE_BLOCK_SCALAR.
func blockScalar(code string) bool {
	for i := 0; i < len(code); i++ {
		if code[i] != ':' {
			continue
		}
		j := chars.SpaceRun(code, i+1)
		if j >= len(code) || code[j] != '>' && code[j] != '|' {
			continue
		}
		k := j + 1
		for k < len(code) && (code[k] >= '0' && code[k] <= '9' || code[k] == '+' || code[k] == '-') {
			k++
		}
		if chars.SpaceRun(code, k) == len(code) {
			return true
		}
	}
	return false
}

// stripLineTrailingCommas removes the comma that ends a line of a multi-line metadata body,
// which block YAML does not allow, as Pele's stripLineTrailingCommas does. Commas inside
// quoted scalars, flow collections and block scalars are content and stay.
func stripLineTrailingCommas(body string) string {
	inSingle, inDouble := false, false
	flowDepth := 0
	blockIndent := -1
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if blockIndent >= 0 {
			if indent := chars.LeadingSpace(line); chars.Trim(line) == "" || indent > blockIndent {
				out = append(out, line)
				continue
			}
			blockIndent = -1
		}
		commentStart := -1
	scan:
		for i := 0; i < len(line); i++ {
			ch := line[i]
			switch {
			case inDouble:
				if ch == '\\' {
					i++
				} else if ch == '"' {
					inDouble = false
				}
			case inSingle:
				if ch == '\'' {
					inSingle = false
				}
			case ch == '"':
				inDouble = true
			case ch == '\'':
				inSingle = true
			case ch == '#' && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t'):
				commentStart = i
				break scan
			case ch == '[' || ch == '{':
				flowDepth++
			case ch == ']' || ch == '}':
				flowDepth = max(0, flowDepth-1)
			}
		}
		if inSingle || inDouble || flowDepth > 0 {
			out = append(out, line)
			continue
		}
		code, comment := line, ""
		if commentStart >= 0 {
			code, comment = line[:commentStart], line[commentStart:]
		}
		if blockScalar(code) {
			blockIndent = chars.LeadingSpace(line)
			out = append(out, line)
			continue
		}
		// code.replace(/,([\t ]*)$/, '$1')
		trimmed := strings.TrimRight(code, " \t")
		if strings.HasSuffix(trimmed, ",") {
			code = trimmed[:len(trimmed)-1] + code[len(trimmed):]
		}
		out = append(out, code+comment)
	}
	return strings.Join(out, "\n")
}

// edgeSemantic is what an arrow of a type and stroke means, "" for nothing, as Pele's
// edgeSemantic gives it.
func edgeSemantic(typ, stroke string) string {
	switch {
	case typ == "arrow_point" && stroke == "normal":
		return "sequence"
	case typ == "arrow_cross" && stroke == "normal":
		return "failure"
	case typ == "arrow_open" && stroke == "dotted":
		return "reference"
	}
	return ""
}

// destructEndLink reads the type, stroke and length of an arrow, or of the end of one, as
// Pele's destructEndLink does.
func destructEndLink(raw string) linkInfo {
	str := chars.Trim(raw)
	line := ""
	last := byte(0)
	if str != "" {
		line = str[:len(str)-1]
		last = str[len(str)-1]
	}
	typ := "arrow_open"
	switch last {
	case 'x':
		typ = "arrow_cross"
	case '>':
		typ = "arrow_point"
	case '-', '.':
		line = str
	}
	stroke := "normal"
	length := len(line) - 1
	if dots := strings.Count(line, "."); dots > 0 {
		stroke = "dotted"
		length = dots
	}
	return linkInfo{Type: typ, Stroke: stroke, Length: length}
}

// semanticMetadata is metadata without the keys that say only how a thing is drawn, nil where
// nothing is left.
// It is Pele's semanticMetadata.
func semanticMetadata(m *yaml.Map) *yaml.Map {
	if m == nil {
		return nil
	}
	out := yaml.NewMap()
	for k, v := range m.All() {
		if !presentationKeys[k] {
			out.Set(k, v)
		}
	}
	if out.Len() == 0 {
		return nil
	}
	return out
}

// syntaxAt is the syntax error of a metadata block that is not valid YAML, at a line and a
// column, which Pele's metadata throws.
func syntaxAt(message string, line, column int) *diagram.Error {
	return &diagram.Error{Code: diagram.Syntax, Message: message, Type: "agentflow", Line: line, Column: column}
}

// metadata reads a metadata block, as Pele's metadata does: a block of one line as a flow
// mapping, and one of more lines as block YAML, without the commas that end its lines if it
// does not read with them.
func (d *DB) metadata(id, source string, loc *loc) (yaml.Value, error) {
	multiline := strings.Contains(source, "\n")
	var src string
	if multiline {
		src = source + "\n"
	} else {
		src = "{\n" + source + "\n}"
	}
	parsed, err := yaml.Parse(src)
	if err != nil {
		stripped := source
		if multiline {
			stripped = stripLineTrailingCommas(source)
		}
		ok := false
		if stripped != source {
			if retried, err2 := yaml.Parse(stripped + "\n"); err2 == nil {
				parsed, ok = retried, true
			}
		}
		if !ok {
			at := ""
			line, column := d.errorAt(loc)
			if loc != nil {
				at = " (" + strconv.Itoa(line) + ":" + strconv.Itoa(column) + ")"
			}
			return nil, syntaxAt(`Metadata of "`+id+`" is not valid YAML. `+err.Error()+at, line, column)
		}
	}
	if parsed == nil {
		parsed = yaml.NewMap()
	}
	return stripPrototypeKeys(parsed), nil
}

// errorAt is the line and column of an error at loc, as Pele's errorAt gives them: located in
// Source as common.Locate locates it, so they are those of the text as its author wrote it.
// Without Source, the line counts the lines of front matter before the text.
func (d *DB) errorAt(loc *loc) (line, column int) {
	if loc == nil {
		return 0, 0
	}
	source := d.options.Source
	if source == "" {
		return loc.FirstLine + d.LineOffset, loc.FirstColumn + 1
	}
	// The start of the line, where indexOf's -1 for a line past the last starts over at 0.
	start := 0
	for k := 1; k < loc.FirstLine; k++ {
		if nl := strings.IndexByte(source[start:], '\n'); nl >= 0 {
			start += nl + 1
		} else {
			start = 0
		}
	}
	// The column is counted in code units.
	at := start
	for n := loc.FirstColumn; n > 0 && at < len(source); n-- {
		r, size := utf8.DecodeRuneInString(source[at:])
		if r > 0xffff {
			n--
		}
		at += size
	}
	l := common.Locate(source, at)
	return l.Line, l.Column
}

// shapeError is the error of a metadata block's unknown shape, at the place of the block, as
// Pele's shapeError makes it.
func (d *DB) shapeError(message string, loc *loc) *diagram.Error {
	e := &diagram.Error{Code: diagram.Semantic, Message: message, Type: "agentflow"}
	e.Line, e.Column = d.errorAt(loc)
	return e
}

// jsonString is JSON.stringify of a value read from YAML.
// Pele's addVertex calls JSON.stringify for an authored shape that is not a string.
func jsonString(v yaml.Value) string {
	var b strings.Builder
	var write func(v yaml.Value)
	write = func(v yaml.Value) {
		switch x := v.(type) {
		case nil:
			b.WriteString("null")
		case bool:
			b.WriteString(strconv.FormatBool(x))
		case float64:
			s := chars.NumberString(x)
			if s == "NaN" || s == "Infinity" || s == "-Infinity" {
				s = "null"
			}
			b.WriteString(s)
		case string:
			b.WriteByte('"')
			for _, r := range x {
				switch r {
				case '"':
					b.WriteString(`\"`)
				case '\\':
					b.WriteString(`\\`)
				case '\b':
					b.WriteString(`\b`)
				case '\f':
					b.WriteString(`\f`)
				case '\n':
					b.WriteString(`\n`)
				case '\r':
					b.WriteString(`\r`)
				case '\t':
					b.WriteString(`\t`)
				default:
					if r < 0x20 {
						b.WriteString(`\u00`)
						b.WriteByte("0123456789abcdef"[r>>4])
						b.WriteByte("0123456789abcdef"[r&15])
					} else {
						b.WriteRune(r)
					}
				}
			}
			b.WriteByte('"')
		case []yaml.Value:
			b.WriteByte('[')
			for i, item := range x {
				if i > 0 {
					b.WriteByte(',')
				}
				write(item)
			}
			b.WriteByte(']')
		case *yaml.Map:
			b.WriteByte('{')
			i := 0
			for k, item := range x.All() {
				if i > 0 {
					b.WriteByte(',')
				}
				i++
				write(k)
				b.WriteByte(':')
				write(item)
			}
			b.WriteByte('}')
		}
	}
	write(v)
	return b.String()
}

// addVertex adds a node, or adds to it, as Pele's addVertex does; metadata is the text of a
// metadata block, if hasMetadata, and loc where it is. text is nil, typ "", and style and
// classes nil where Pele's are undefined. Metadata on the id of a flow, a connector or an
// edge is theirs.
func (d *DB) addVertex(id string, text *flowText, typ string, style, classes []string, props []Prop, metadata string, hasMetadata bool, loc *loc) error {
	if id == "" || chars.Trim(id) == "" {
		return nil
	}

	var doc yaml.Value
	var fields *yaml.Map
	authoredShape, hasAuthored := "", false
	if hasMetadata {
		v, err := d.metadata(id, metadata, loc)
		if err != nil {
			return err
		}
		doc = v
		if m, ok := v.(*yaml.Map); ok {
			fields = m
			if s, ok := m.Value("shape").(string); ok {
				authoredShape, hasAuthored = s, true
				m.Set("shape", resolveShapeAlias(s))
			}
		}
	}
	truthy := hasMetadata && yaml.Truthy(doc)

	if sg := d.subgraphLookup[id]; sg != nil && truthy {
		sg.Metadata = merge(sg.Metadata, doc)
		return nil
	}
	if c := d.connectors[id]; c != nil && truthy {
		c.Metadata = merge(c.Metadata, doc)
		return nil
	}
	// Mermaid reserves this id and creates no node for it.
	if id == "connectors" {
		return nil
	}
	if edge := d.edgeByID[id]; edge != nil {
		if v, ok := fields.Get("animate"); ok {
			edge.Animate = v
		}
		if v, ok := fields.Get("animation"); ok {
			edge.Animation = v
		}
		if v, ok := fields.Get("curve"); ok {
			edge.Interpolate = v
		}
		if truthy {
			edge.Metadata = merge(edge.Metadata, doc)
		}
		return nil
	}

	vertex := d.Node(id)
	created := vertex == nil
	if created {
		vertex = &Node{ID: id, LabelType: LabelText, Styles: []string{}, Classes: []string{}}
		d.setNode(vertex)
	}
	if text != nil {
		vertex.LabelType = text.Type
		vertex.Text = unquote(chars.Trim(text.Text))
	} else if created {
		vertex.Text = id
	}
	if typ != "" {
		vertex.Type = typ
	}
	vertex.Styles = append(vertex.Styles, style...)
	vertex.Classes = append(vertex.Classes, classes...)
	if vertex.Props == nil {
		vertex.Props = append([]Prop{}, props...)
	} else {
		for _, p := range props {
			vertex.Props = assignProp(vertex.Props, p)
		}
	}

	if !truthy {
		return nil
	}
	vertex.Metadata = merge(vertex.Metadata, doc)
	if fields == nil {
		return nil
	}

	if shape := fields.Value("shape"); yaml.Truthy(shape) {
		var authored yaml.Value = shape
		if hasAuthored {
			authored = authoredShape
		}
		name, ok := authored.(string)
		if !ok {
			return d.shapeError("No such shape: "+jsonString(authored)+".", loc)
		}
		if name != chars.ToLower(name) || strings.Contains(name, "_") {
			return d.shapeError("No such shape: "+name+". Shape names should be lowercase.", loc)
		}
		resolved := shape.(string)
		if !isValidShape(resolved) {
			return d.shapeError("No such shape: "+resolved+".", loc)
		}
		vertex.Type = resolved
	}
	if v := fields.Value("label"); yaml.Truthy(v) {
		vertex.Text = yaml.String(v)
		t, _ := fields.Value("labelType").(string)
		vertex.LabelType = labelType(t)
	}
	return nil
}

// assignProp sets a property as Object.assign does: a __proto__ that the props do not hold
// already is the prototype's, and setting it to a string does nothing.
// It is Pele's addVertex adding props to a node's.
func assignProp(props []Prop, p Prop) []Prop {
	for i := range props {
		if props[i].Key == p.Key {
			props[i].Value = p.Value
			return props
		}
	}
	if p.Key == "__proto__" {
		return props
	}
	return append(props, p)
}

// agentError is an error of an agent flow with a code, as Pele's database throws a PeleError
// of type agentflow.
func agentError(message string, code diagram.Code) *diagram.Error {
	return common.NewError(message, code, "agentflow")
}

// addSingleLink adds an edge, as Pele's addSingleLink does; id is the arrow's own, if hasID.
func (d *DB) addSingleLink(start, end string, info *linkInfo, id string, hasID bool) error {
	edge := &Edge{Start: start, End: end, LabelType: LabelText, Classes: []string{}}
	if d.DefaultInterpolate != "" {
		edge.Interpolate = d.DefaultInterpolate
	}
	if info != nil {
		if info.Text != nil {
			edge.Text = unquote(chars.Trim(info.Text.Text))
			edge.LabelType = labelType(string(info.Text.Type))
		}
		edge.Type = info.Type
		edge.Stroke = info.Stroke
		edge.Length = min(info.Length, 10)
		edge.EdgeSemantic = info.EdgeSemantic
	}
	pair := start + "\x00" + end
	existing := d.pairCount[pair]
	if _, taken := d.edgeByID[id]; hasID && id != "" && !taken {
		edge.ID = id
		edge.IsUserDefinedID = true
	} else {
		n := 0
		if existing != 0 {
			n = existing + 1
		}
		edge.ID = "L_" + start + "_" + end + "_" + strconv.Itoa(n)
	}

	if max := d.options.MaxEdges; max >= 0 && len(d.Edges) >= max {
		return agentError("Edge limit exceeded. The limit is "+strconv.Itoa(max)+".", diagram.Limit)
	}
	d.Edges = append(d.Edges, edge)
	d.pairCount[pair] = existing + 1
	if _, ok := d.edgeByID[edge.ID]; !ok {
		d.edgeByID[edge.ID] = edge
	}
	return nil
}

// addLink adds an edge from each of starts to each of ends, as Pele's addLink does. The id the
// arrow gives goes to the edge from the last start to the first end.
func (d *DB) addLink(starts, ends []string, info *linkInfo) error {
	id, hasID := "", false
	if info != nil && info.HasID {
		id, hasID = strings.Replace(info.ID, "@", "", 1), true
	}
	lastStart := starts[len(starts)-1]
	firstEnd := ends[0]
	for _, start := range starts {
		for _, end := range ends {
			mine := hasID && start == lastStart && end == firstEnd
			if err := d.addSingleLink(start, end, info, id, mine); err != nil {
				return err
			}
		}
	}
	return nil
}

// updateLinkInterpolate sets the curve of the edges at positions, or of all with "default".
// It does as Pele's updateLinkInterpolate does.
func (d *DB) updateLinkInterpolate(positions []string, interpolate string) error {
	for _, pos := range positions {
		if pos == "default" {
			d.DefaultInterpolate = interpolate
			continue
		}
		edge, err := d.edgeAt(pos)
		if err != nil {
			return err
		}
		edge.Interpolate = interpolate
	}
	return nil
}

// updateLink sets the style of the edges at positions, or of all with "default", as Pele's
// updateLink does. Pele gives every edge the same array and adds fill:none to it once, which
// all of them then see.
func (d *DB) updateLink(positions []string, style []string) error {
	var styled []*Edge
	isDefault := false
	for _, pos := range positions {
		if pos == "default" {
			isDefault = true
			continue
		}
		edge, err := d.edgeAt(pos)
		if err != nil {
			return err
		}
		styled = append(styled, edge)
		if len(style) > 0 && !hasFill(style) {
			style = append(style, "fill:none")
		}
	}
	if isDefault {
		d.DefaultEdgeStyle = style
	}
	for _, e := range styled {
		e.Style = style
	}
	return nil
}

// hasFill reports whether a style has a fill, the test of Pele's updateLink for adding
// fill:none.
func hasFill(style []string) bool {
	for _, s := range style {
		if strings.HasPrefix(s, "fill") {
			return true
		}
	}
	return false
}

// edgeAt is the edge at a position linkStyle gives. Mermaid checks the bounds only for a
// numeric index and crashes on the strings its own parser passes.
// It is Pele's edgeAt.
func (d *DB) edgeAt(pos string) (*Edge, error) {
	n := chars.Number(pos)
	if n >= 0 && n < float64(len(d.Edges)) && n == float64(int(n)) {
		return d.Edges[int(n)], nil
	}
	return nil, agentError("The index "+pos+" for linkStyle is out of bounds. Valid indices for linkStyle are between 0 and "+
		strconv.Itoa(len(d.Edges)-1)+". (Help: Ensure that the index is within the range of existing edges.)", diagram.Semantic)
}

// addClass defines classes, or adds to them, as Pele's addClass does: ids are separated by
// commas, and so are styles, but for an escaped one.
func (d *DB) addClass(ids string, styles []string) {
	joined := strings.Join(styles, ",")
	joined = strings.ReplaceAll(joined, "\\,", "§§§")
	joined = strings.ReplaceAll(joined, ",", ";")
	joined = strings.ReplaceAll(joined, "§§§", ",")
	style := strings.Split(joined, ";")
	for _, id := range strings.Split(ids, ",") {
		class := d.classes[id]
		if class == nil {
			class = &ClassDef{ID: id, Styles: []string{}, TextStyles: []string{}}
			d.classes[id] = class
			d.Classes = append(d.Classes, class)
		}
		for _, s := range style {
			if strings.Contains(s, "color") {
				class.TextStyles = append(class.TextStyles, strings.Replace(s, "fill", "bgFill", 1))
			}
			class.Styles = append(class.Styles, s)
		}
	}
}

// setDirection sets the diagram's direction, reading Mermaid's arrows for it.
// It does as Pele's setDirection does.
func (d *DB) setDirection(dir string) {
	s := chars.Trim(dir)
	if strings.Contains(s, "<") {
		s = "RL"
	}
	if strings.Contains(s, "^") {
		s = "BT"
	}
	if strings.Contains(s, ">") {
		s = "LR"
	}
	if strings.Contains(s, "v") {
		s = "TB"
	}
	if s == "TD" {
		s = "TB"
	}
	d.Direction = s
}

// setClass gives the nodes, edges and flows of ids, separated by commas, a class.
// It does as Pele's setClass does.
func (d *DB) setClass(ids, className string) {
	for _, id := range strings.Split(ids, ",") {
		if n := d.Node(id); n != nil {
			n.Classes = append(n.Classes, className)
		}
		if e := d.edgeByID[id]; e != nil {
			e.Classes = append(e.Classes, className)
		}
		if s := d.subgraphLookup[id]; s != nil {
			s.Classes = append(s.Classes, className)
		}
	}
}

// setTooltip gives the nodes of ids a tooltip.
// It does as Pele's setTooltip does.
func (d *DB) setTooltip(ids, tooltip string) {
	for _, id := range strings.Split(ids, ",") {
		if i, ok := d.tooltips[id]; ok {
			d.Tooltips[i].Text = tooltip
			continue
		}
		d.tooltips[id] = len(d.Tooltips)
		d.Tooltips = append(d.Tooltips, Tooltip{id, tooltip})
	}
}

// setLink makes the nodes of ids links to a URL, checked as Pele's safeUrl checks it.
// It does as Pele's setLink does.
func (d *DB) setLink(ids, link, target string) {
	u := chars.Trim(link)
	for _, id := range strings.Split(ids, ",") {
		if v := d.Node(id); v != nil {
			v.Link = ""
			if u != "" {
				v.Link = url.SafeURL(u)
			}
			v.LinkTarget = target
		}
	}
	d.setClass(ids, "clickable")
}

// setClickEvent marks the nodes of ids clickable. Callbacks are never run, as in Mermaid's
// strict mode.
// It does as Pele's setClickEvent does.
func (d *DB) setClickEvent(ids string) { d.setClass(ids, "clickable") }

// setAccTitle sets the accessible title, without the white space it starts with.
// It does as Pele's setAccTitle does.
func (d *DB) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, without the white space each of its
// lines but the first starts with: text.replace(/\n\s+/g, '\n').
// It does as Pele's setAccDescription does.
func (d *DB) setAccDescription(text string) {
	d.AccDescr = chars.DropIndents(text)
}

// docItem is what a statement of a flow's body gives it, Pele's DocItem: a string, a list of
// node ids, or a direction.
type docItem struct {
	kind int
	text string
	list []string
}

const (
	itemString = iota
	itemList
	itemDir
)

// addSubGraph adds a flow of what its body gave, or adds to the one of the same id, as Pele's
// addSubGraph does, and gives its id. idText and titleText are nil for a flow with neither.
func (d *DB) addSubGraph(idText *flowText, list []docItem, titleText *flowText) string {
	id, hasID := "", idText != nil
	if hasID {
		id = chars.Trim(idText.Text)
	}
	title := ""
	if titleText != nil {
		title = titleText.Text
	}

	seen := map[string]bool{}
	var nodeList []string
	dir, hasDir := "", false
	add := func(s string) {
		if chars.Trim(s) != "" && !seen[s] {
			seen[s] = true
			nodeList = append(nodeList, s)
		}
	}
	for _, entry := range list {
		switch entry.kind {
		case itemString:
			add(entry.text)
		case itemList:
			for _, item := range entry.list {
				add(item)
			}
		default:
			dir, hasDir = entry.text, true
		}
	}
	if !hasDir && d.options.InheritDir {
		dir = d.Direction
		if dir == "" {
			dir = d.options.Direction
		}
	}

	if !hasID {
		id = "subGraph" + strconv.Itoa(d.subCount)
	}
	d.subCount++

	// A flow never contains itself, a globally scoped node, or a node another flow already
	// holds.
	nodes := []string{}
	for _, nodeID := range nodeList {
		if nodeID != id && !d.globalNodes[nodeID] && !d.members[nodeID] {
			nodes = append(nodes, nodeID)
		}
	}
	for _, nodeID := range nodes {
		d.members[nodeID] = true
	}

	if existing := d.subgraphLookup[id]; existing != nil {
		// A repeated id adds its members to the first flow; its title and direction win.
		existing.Nodes = append(existing.Nodes, nodes...)
		if t := chars.Trim(title); t != "" {
			existing.Title = t
		}
		if dir != "" {
			existing.Dir = dir
		}
		return id
	}
	lt := LabelMarkdown
	if titleText != nil {
		lt = labelType(string(titleText.Type))
	}
	sg := &Subgraph{ID: id, Nodes: nodes, Title: chars.Trim(title), Classes: []string{}, Dir: dir, LabelType: lt, Type: "flow"}
	d.Subgraphs = append(d.Subgraphs, sg)
	d.subgraphLookup[id] = sg
	return id
}

// addGlobal keeps every node named inside a `global` block out of the flows that mention it,
// as Pele's addGlobal does.
func (d *DB) addGlobal(list []docItem) {
	release := false
	// add keeps a node out of the flows, and notes whether one held it, as Pele's add does.
	add := func(item string) {
		id := chars.Trim(item)
		if id == "" {
			return
		}
		d.globalNodes[id] = true
		if d.members[id] {
			delete(d.members, id)
			release = true
		}
	}
	for _, entry := range list {
		switch entry.kind {
		case itemString:
			add(entry.text)
		case itemList:
			for _, item := range entry.list {
				add(item)
			}
		}
	}
	if !release {
		return
	}
	for _, sg := range d.Subgraphs {
		kept := []string{}
		for _, nodeID := range sg.Nodes {
			if !d.globalNodes[nodeID] {
				kept = append(kept, nodeID)
			}
		}
		sg.Nodes = kept
	}
}

// addConnector declares a connector, or retitles it, as Pele's addConnector does, and gives
// its id: "" for one with none.
func (d *DB) addConnector(textObj, titleObj *flowText) string {
	id := ""
	if textObj != nil {
		id = chars.Trim(textObj.Text)
	}
	if id == "" {
		return ""
	}
	title := ""
	if titleObj != nil {
		title = chars.Trim(titleObj.Text)
	}
	if connector := d.connectors[id]; connector == nil {
		created := &Node{ID: id, LabelType: LabelText, Styles: []string{}, Classes: []string{}, IsConnector: true, Text: id}
		if title != "" {
			created.Text = title
		}
		d.connectors[id] = created
		d.Connectors = append(d.Connectors, created)
		d.setNode(created)
	} else if title != "" {
		connector.Text = title
	}
	return id
}

// destructLink reads an arrow, as Pele's destructLink does: end is the arrow, or its end where
// start is the start of an arrow with a label between. An arrow whose start and end do not
// agree on being dotted is of type and stroke "INVALID".
func (d *DB) destructLink(end, start string) linkInfo {
	info := destructEndLink(end)
	if start != "" {
		// The first half of a split arrow only says whether the line is dotted.
		stroke := "normal"
		if strings.Contains(start, ".") {
			stroke = "dotted"
		}
		if stroke != info.Stroke {
			return linkInfo{Type: "INVALID", Stroke: "INVALID"}
		}
	}
	info.EdgeSemantic = edgeSemantic(info.Type, info.Stroke)
	return info
}

// VertexKind is what a node is, from its shape, as Pele's vertexKind gives it: "tool",
// "action", "input", "refdoc", "decision" or "task".
func (d *DB) VertexKind(v *Node) string {
	if isToolDefinition(v) {
		return "tool"
	}
	switch resolveShapeAlias(v.Type) {
	case "hexagon", "hex":
		return "action"
	case "lean-right", "lean_right":
		return "input"
	case "lin-doc", "lined-document":
		return "refdoc"
	case "diamond":
		return "decision"
	}
	return "task"
}

// SemanticVertex is a node as the semantic model has it.
type SemanticVertex struct {
	ID, Label, Shape, VertexKind string
	Metadata                     *yaml.Map
}

// SemanticEdge is an edge as the semantic model has it; a string left out is "".
type SemanticEdge struct {
	Start, End, ID, Label, Type, Stroke, EdgeSemantic string
	Length                                            int
	Metadata                                          *yaml.Map
}

// SemanticSubgraph is a flow as the semantic model has it.
type SemanticSubgraph struct {
	ID, Type, Title string
	Nodes           []string
	Metadata        *yaml.Map
	Direction       string
}

// SemanticConnector is a connector as the semantic model has it; Title is "" where it is
// the id.
type SemanticConnector struct {
	ID, Title string
	Metadata  *yaml.Map
}

// SemanticModel is the diagram with presentation-only controls left out, for tools that
// read what it means: Pele's SemanticModel.
type SemanticModel struct {
	Direction   string
	Vertices    []SemanticVertex
	Edges       []SemanticEdge
	Subgraphs   []SemanticSubgraph
	Connectors  []SemanticConnector
	Diagnostics []*Diagnostic
}

// SemanticModel is what the diagram means, as Pele's semanticModel gives it.
func (d *DB) SemanticModel() *SemanticModel {
	m := &SemanticModel{Direction: d.Direction, Diagnostics: d.Diagnostics}
	for _, v := range d.Nodes {
		if d.subgraphLookup[v.ID] != nil {
			continue
		}
		metadata := semanticMetadata(v.Metadata)
		if v.IsConnector {
			c := SemanticConnector{ID: v.ID, Metadata: metadata}
			if v.Text != v.ID {
				c.Title = v.Text
			}
			m.Connectors = append(m.Connectors, c)
			continue
		}
		m.Vertices = append(m.Vertices, SemanticVertex{ID: v.ID, Label: v.Text, Shape: resolveShapeAlias(v.Type),
			VertexKind: d.VertexKind(v), Metadata: metadata})
	}
	for _, e := range d.Edges {
		se := SemanticEdge{Start: e.Start, End: e.End, ID: e.ID, Label: e.Text, Type: e.Type, Stroke: e.Stroke,
			EdgeSemantic: e.EdgeSemantic, Length: e.Length}
		if e.Metadata.Len() > 0 {
			se.Metadata = e.Metadata.Spread(nil)
		}
		m.Edges = append(m.Edges, se)
	}
	for _, sg := range d.Subgraphs {
		m.Subgraphs = append(m.Subgraphs, SemanticSubgraph{ID: sg.ID, Nodes: append([]string{}, sg.Nodes...), Type: sg.Type,
			Title: sg.Title, Direction: sg.Dir, Metadata: semanticMetadata(sg.Metadata)})
	}
	return m
}

// position is where a span of the source is, with the lines of the front matter counted, as
// Pele's position gives it: a statement with no span is at line 0 of the text, column 0.
func (d *DB) position(loc *loc) *ElementPosition {
	if loc == nil {
		return &ElementPosition{StartLine: d.LineOffset, EndLine: d.LineOffset}
	}
	return &ElementPosition{
		StartLine:   loc.FirstLine + d.LineOffset,
		StartColumn: loc.FirstColumn,
		EndLine:     loc.LastLine + d.LineOffset,
		EndColumn:   loc.LastColumn,
	}
}

// pushMapping records where a statement is, and keeps the first record of each id for the
// diagnostics about it, as Pele's pushMapping does.
func (d *DB) pushMapping(id, typ string, position *ElementPosition) *ElementMapping {
	m := &ElementMapping{ID: id, Type: typ, Position: position}
	d.Mappings = append(d.Mappings, m)
	if _, ok := d.firstMapping[id]; !ok {
		d.firstMapping[id] = m
	}
	return m
}

// addVertexMapping records where a node is written. A bare reference becomes an attachment
// if a metadata block follows it.
// It is Pele's addVertexMapping, which is told the shape where this is told whether there is
// none.
func (d *DB) addVertexMapping(id string, bare bool, loc *loc) {
	if id == "" {
		return
	}
	m := d.pushMapping(id, StatementVertex, d.position(loc))
	if bare {
		d.bareMappings[m] = true
	}
}

// extendVertexMapping widens a node's span over the metadata block that follows it.
// It is Pele's extendVertexMapping.
func (d *DB) extendVertexMapping(id string, loc *loc) {
	if id == "" || loc == nil {
		return
	}
	end := d.position(loc)
	for i := len(d.Mappings) - 1; i >= 0; i-- {
		m := d.Mappings[i]
		if m.Type != StatementVertex || m.ID != id {
			continue
		}
		if d.bareMappings[m] {
			delete(d.bareMappings, m)
			m.Type = StatementAttachment
		}
		p := m.Position
		if end.EndLine > p.EndLine || end.EndLine == p.EndLine && end.EndColumn > p.EndColumn {
			p.EndLine = end.EndLine
			p.EndColumn = end.EndColumn
		}
		return
	}
	d.pushMapping(id, StatementAttachment, end)
}

// addEdgeMapping records where an edge statement is, under the ids of the nodes it ends at,
// as Pele's addEdgeMapping does.
func (d *DB) addEdgeMapping(toNodes []string, loc *loc) {
	var ids []string
	for _, n := range toNodes {
		if n != "" {
			ids = append(ids, n)
		}
	}
	id := "edge"
	if len(ids) > 0 {
		id = strings.Join(ids, ">")
	}
	d.pushMapping(id, StatementEdge, d.position(loc))
}

// blockMapping records where a block is, from its keyword to its end, as Pele's blockMapping
// does.
func (d *DB) blockMapping(id, typ string, startLoc, endLoc *loc) {
	if id == "" {
		return
	}
	start := d.position(startLoc)
	end := start
	if endLoc != nil {
		end = d.position(endLoc)
	}
	d.pushMapping(id, typ, &ElementPosition{StartLine: start.StartLine, StartColumn: start.StartColumn,
		EndLine: end.EndLine, EndColumn: end.EndColumn})
}

// addSubgraphMapping records where a flow is, as Pele's addSubgraphMapping does.
func (d *DB) addSubgraphMapping(id *flowText, startLoc, endLoc *loc) {
	text := ""
	if id != nil {
		text = id.Text
	}
	d.blockMapping(text, StatementSubgraph, startLoc, endLoc)
}

// addConnectorMapping records where a connector is, as Pele's addConnectorMapping does.
func (d *DB) addConnectorMapping(id *flowText, startLoc, endLoc *loc) {
	text := ""
	if id != nil {
		text = chars.Trim(id.Text)
	}
	d.blockMapping(text, StatementConnector, startLoc, endLoc)
}

// emit records a diagnostic about a node or an edge, once however often the graph is built.
// It is Pele's emit.
func (d *DB) emit(id, severity, message, nodeID, edgeID string) {
	anchor := nodeID
	if anchor == "" {
		anchor = edgeID
	}
	diag := &Diagnostic{ID: id, Severity: severity, Message: message}
	if nodeID != "" {
		diag.NodeID = nodeID
	} else if edgeID != "" {
		diag.EdgeID = edgeID
	}
	if anchor != "" {
		if m := d.firstMapping[anchor]; m != nil {
			diag.Position = m.Position
		}
	}
	same := d.reported[anchor]
	for _, o := range same {
		if o.ID == id && o.Severity == severity && o.Message == message && o.NodeID == diag.NodeID && o.EdgeID == diag.EdgeID {
			return
		}
	}
	d.reported[anchor] = append(same, diag)
	d.Diagnostics = append(d.Diagnostics, diag)
	if d.options.Log != nil {
		d.options.Log(severity, "agentflow["+id+"]: "+message)
	}
}

// emitWarning records a warning about the node of nodeID.
// It is Pele's emitWarning.
func (d *DB) emitWarning(id, message, nodeID string) { d.emit(id, "warning", message, nodeID, "") }

// emitError records an error about the node of nodeID.
// It is Pele's emitError.
func (d *DB) emitError(id, message, nodeID string) { d.emit(id, "error", message, nodeID, "") }
