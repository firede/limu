package flowchart

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
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

// Text is a label as written, with how it is read. Pele's FlowText.
type Text struct {
	Text string
	Type LabelType
}

// Prop is a property a node is given with `id[|name:value|label]`.
type Prop struct {
	Key, Value string
}

// Node is a node of a flowchart, Pele's FlowNode. A string that Pele may leave undefined is
// empty here where it is; no reader of a node tells the two apart.
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
	Icon       string
	Form       string
	Pos        string
	Img        string
	Constraint string
	// AssetWidth and AssetHeight are the size of a node's image or icon, nil where unset; NaN
	// where set to what is not a number.
	AssetWidth  *float64
	AssetHeight *float64
}

// Edge is an edge of a flowchart, Pele's FlowEdge.
type Edge struct {
	Start     string
	End       string
	Type      string
	Text      string
	LabelType LabelType
	Stroke    string
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
}

// ClassDef is a class of styles, from classDef. Pele's FlowClassDef.
type ClassDef struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// Subgraph is a subgraph, Pele's FlowSubgraph.
type Subgraph struct {
	ID        string
	Nodes     []string
	Title     string
	Classes   []string
	Dir       string
	LabelType LabelType
	// Metadata is what metadata blocks give the subgraph, nil where none do.
	Metadata *yaml.Map
}

// Tooltip is a node's tooltip, from click.
type Tooltip struct {
	ID, Text string
}

// LinkInfo is what an arrow says of an edge: Pele's LinkInfo.
type LinkInfo struct {
	Type   string
	Stroke string
	Length int
	Text   *Text
	ID     string
}

// Options are Pele's FlowDbOptions.
type Options struct {
	// MaxEdges is the most edges the flowchart may have; no limit when it is negative.
	MaxEdges int
	// InheritDir gives a subgraph that sets no direction the flowchart's.
	InheritDir bool
	// Direction is the direction InheritDir gives where the flowchart has none.
	Direction string
	// Warn, if set, is told of what may be a mistake: a style for a node not yet seen.
	Warn func(message string)
}

// DB is what a flowchart's text says, as Pele's FlowDb holds it: its nodes, edges, classes,
// subgraphs and the rest, in the order they were first seen.
type DB struct {
	Direction string
	// Nodes are the nodes in the order they were first seen; Node finds one by its id.
	Nodes     []*Node
	Edges     []*Edge
	Subgraphs []*Subgraph
	// Classes are the classes in the order they were first defined; Class finds one.
	Classes []*ClassDef
	// Tooltips are the tooltips in the order their nodes were first given one; Tooltip finds
	// one.
	Tooltips []Tooltip
	// DefaultEdgeStyle is what `linkStyle default` sets, nil where it is not set.
	DefaultEdgeStyle []string
	// DefaultInterpolate is what `linkStyle default interpolate` sets, "" where it is not set.
	DefaultInterpolate string
	Title              string
	AccTitle           string
	AccDescr           string

	nodes          map[string]*Node
	classes        map[string]*ClassDef
	tooltips       map[string]int
	subgraphLookup map[string]*Subgraph
	members        map[string]bool
	edgeByID       map[string]*Edge
	pairCount      map[string]int
	subCount       int
	options        Options
}

// NewDB is an empty flowchart.
// It is the constructor of Pele's FlowDb.
func NewDB(options Options) *DB {
	return &DB{
		nodes:          map[string]*Node{},
		classes:        map[string]*ClassDef{},
		tooltips:       map[string]int{},
		subgraphLookup: map[string]*Subgraph{},
		members:        map[string]bool{},
		edgeByID:       map[string]*Edge{},
		pairCount:      map[string]int{},
		options:        options,
	}
}

// Node is the node of id, or nil.
// It is Pele's nodes.get(id).
func (d *DB) Node(id string) *Node { return d.nodes[id] }

// Class is the class of id, or nil.
// It is Pele's classes.get(id).
func (d *DB) Class(id string) *ClassDef { return d.classes[id] }

// Tooltip is the tooltip of the node of id, and whether it has one.
// It is Pele's tooltips.get(id): Pele keeps the tooltips in a Map, which limu keeps as a
// list and an index.
func (d *DB) Tooltip(id string) (string, bool) {
	i, ok := d.tooltips[id]
	if !ok {
		return "", false
	}
	return d.Tooltips[i].Text, true
}

// Subgraph is the subgraph of id, or nil.
// It is Pele's subgraph.
func (d *DB) Subgraph(id string) *Subgraph { return d.subgraphLookup[id] }

// labelType is a label type as given, if it is one, and "markdown" otherwise, as Pele's
// labelType gives it.
func labelType(t string) LabelType {
	switch t {
	case "markdown", "string", "text":
		return LabelType(t)
	}
	return LabelMarkdown
}

// unquote drops the double quotes text starts and ends with, if it has both, as Pele's
// unquote does.
func unquote(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

// destructStartLink reads the start of an arrow that has a label between its start and its
// end, its head and its stroke, as Pele's destructStartLink does.
func destructStartLink(raw string) LinkInfo {
	str := chars.Trim(raw)
	typ := "arrow_open"
	if str != "" {
		switch str[0] {
		case '<':
			typ = "arrow_point"
			str = str[1:]
		case 'x':
			typ = "arrow_cross"
			str = str[1:]
		case 'o':
			typ = "arrow_circle"
			str = str[1:]
		}
	}
	stroke := "normal"
	if strings.Contains(str, "=") {
		stroke = "thick"
	}
	if strings.Contains(str, ".") {
		stroke = "dotted"
	}
	return LinkInfo{Type: typ, Stroke: stroke}
}

// destructEndLink reads an arrow, or the end of one with a label between, its heads, stroke
// and length, as Pele's destructEndLink does; Pele's countChar is strings.Count here.
func destructEndLink(raw string) LinkInfo {
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
		if strings.HasPrefix(str, "x") {
			typ = "double_" + typ
			line = line[1:]
		}
	case '>':
		typ = "arrow_point"
		if strings.HasPrefix(str, "<") {
			typ = "double_" + typ
			line = line[1:]
		}
	case 'o':
		typ = "arrow_circle"
		if strings.HasPrefix(str, "o") {
			typ = "double_" + typ
			line = line[1:]
		}
	}
	stroke := "normal"
	length := len(line) - 1
	if strings.HasPrefix(line, "=") {
		stroke = "thick"
	}
	if strings.HasPrefix(line, "~") {
		stroke = "invisible"
	}
	if dots := strings.Count(line, "."); dots > 0 {
		stroke = "dotted"
		length = dots
	}
	return LinkInfo{Type: typ, Stroke: stroke, Length: length}
}

// flowError is an error of a flowchart, the PeleError of type flowchart that Pele's FlowDb
// throws.
func flowError(message string, code diagram.Code) *diagram.Error {
	return diagram.NewError(message, code, "flowchart")
}

// addVertex adds a node, or adds to it, as Pele's addVertex does. text is nil, typ "", and
// style and classes nil where Pele's are undefined; metadata is the text of a metadata block,
// if hasMetadata.
func (d *DB) addVertex(id string, text *Text, typ string, style, classes []string, dir string, props []Prop, metadata string, hasMetadata bool) error {
	if id == "" || chars.Trim(id) == "" {
		return nil
	}

	var doc *yaml.Map
	if hasMetadata {
		var src string
		if strings.Contains(metadata, "\n") {
			src = metadata + "\n"
		} else {
			src = "{\n" + metadata + "\n}"
		}
		parsed, err := yaml.Parse(src)
		if err != nil {
			return flowError(`Metadata of "`+id+`" is not valid YAML. `+err.Error(), diagram.Syntax)
		}
		doc, _ = parsed.(*yaml.Map)
	}

	if sg := d.subgraphLookup[id]; sg != nil && doc != nil {
		sg.Metadata = sg.Metadata.Spread(doc)
		return nil
	}

	if edge := d.edgeByID[id]; edge != nil {
		if v, ok := doc.Get("animate"); ok {
			edge.Animate = v
		}
		if v, ok := doc.Get("animation"); ok {
			edge.Animation = v
		}
		if v, ok := doc.Get("curve"); ok {
			edge.Interpolate = v
		}
		return nil
	}

	vertex := d.nodes[id]
	created := vertex == nil
	if created {
		if text == nil && typ == "" && style != nil && d.options.Warn != nil {
			d.options.Warn(`Style applied to unknown node "` + id + `". This may indicate a typo. The node will be created automatically.`)
		}
		vertex = &Node{ID: id, LabelType: LabelText, Styles: []string{}, Classes: []string{}}
		d.nodes[id] = vertex
		d.Nodes = append(d.Nodes, vertex)
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
	if dir != "" {
		vertex.Dir = dir
	}
	if vertex.Props == nil {
		vertex.Props = append([]Prop{}, props...)
	} else {
		for _, p := range props {
			vertex.Props = assignProp(vertex.Props, p)
		}
	}

	if doc == nil {
		return nil
	}

	if shape := doc.Value("shape"); yaml.Truthy(shape) {
		name := yaml.String(shape)
		if name != chars.ToLower(name) || strings.Contains(name, "_") {
			return flowError("No such shape: "+name+". Shape names should be lowercase.", diagram.Semantic)
		}
		if !IsValidShape(name) {
			return flowError("No such shape: "+name+".", diagram.Semantic)
		}
		vertex.Type = name
	}
	label, isString := doc.Value("label").(string)
	noLabel := !isString || chars.Trim(label) == ""
	if v := doc.Value("label"); yaml.Truthy(v) {
		vertex.Text = yaml.String(v)
		t, _ := doc.Value("labelType").(string)
		vertex.LabelType = labelType(t)
	}
	if v := doc.Value("icon"); yaml.Truthy(v) {
		vertex.Icon = yaml.String(v)
		if noLabel && vertex.Text == id {
			vertex.Text = ""
		}
	}
	if v := doc.Value("form"); yaml.Truthy(v) {
		vertex.Form = yaml.String(v)
	}
	if v := doc.Value("pos"); yaml.Truthy(v) {
		vertex.Pos = yaml.String(v)
	}
	if v := doc.Value("img"); yaml.Truthy(v) {
		vertex.Img = yaml.String(v)
		if noLabel && vertex.Text == id {
			vertex.Text = ""
		}
	}
	if v := doc.Value("constraint"); yaml.Truthy(v) {
		vertex.Constraint = yaml.String(v)
	}
	if v := doc.Value("w"); yaml.Truthy(v) {
		w := yaml.Number(v)
		vertex.AssetWidth = &w
	}
	if v := doc.Value("h"); yaml.Truthy(v) {
		h := yaml.Number(v)
		vertex.AssetHeight = &h
	}
	return nil
}

// assignProp sets a property as Object.assign does: a __proto__ that the props do not hold
// already is the prototype's, and setting it to a string does nothing.
// It is the Object.assign of Pele's addVertex, one property at a time.
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

// addSingleLink adds an edge, as Pele's addSingleLink does; id "" gives it one of its own.
func (d *DB) addSingleLink(start, end string, info *LinkInfo, id string) error {
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
	}
	if _, taken := d.edgeByID[id]; id != "" && !taken {
		edge.ID = id
		edge.IsUserDefinedID = true
	} else {
		existing := d.pairCount[start+"\x00"+end]
		n := 0
		if existing != 0 {
			n = existing + 1
		}
		edge.ID = "L_" + start + "_" + end + "_" + strconv.Itoa(n)
	}

	if max := d.options.MaxEdges; max >= 0 && len(d.Edges) >= max {
		return flowError("Edge limit exceeded. "+strconv.Itoa(len(d.Edges))+" edges found, but the limit is "+strconv.Itoa(max)+".", diagram.Limit)
	}
	d.Edges = append(d.Edges, edge)
	d.pairCount[start+"\x00"+end]++
	if _, ok := d.edgeByID[edge.ID]; !ok {
		d.edgeByID[edge.ID] = edge
	}
	return nil
}

// addLink adds an edge from each of starts to each of ends, as Pele's addLink does. The id the
// arrow gives goes to the edge from the last start to the first end.
func (d *DB) addLink(starts, ends []string, info *LinkInfo) error {
	id := ""
	if info != nil {
		id = strings.Replace(info.ID, "@", "", 1)
	}
	lastStart := starts[len(starts)-1]
	firstEnd := ends[0]
	for _, start := range starts {
		for _, end := range ends {
			edgeID := ""
			if start == lastStart && end == firstEnd {
				edgeID = id
			}
			if err := d.addSingleLink(start, end, info, edgeID); err != nil {
				return err
			}
		}
	}
	return nil
}

// updateLinkInterpolate sets the curve of the edges at positions, or of all with "default".
// It is Pele's updateLinkInterpolate.
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

// hasFill reports whether any of a style's parts sets a fill, the test Pele's updateLink
// makes with style.some.
func hasFill(style []string) bool {
	for _, s := range style {
		if strings.HasPrefix(s, "fill") {
			return true
		}
	}
	return false
}

// edgeAt is the edge at a position linkStyle gives, or an error where there is none, as
// Pele's edgeAt gives it.
func (d *DB) edgeAt(pos string) (*Edge, error) {
	n := chars.Number(pos)
	if n >= 0 && n < float64(len(d.Edges)) && n == float64(int(n)) {
		return d.Edges[int(n)], nil
	}
	return nil, flowError("The index "+pos+" for linkStyle is out of bounds. Valid indices for linkStyle are between 0 and "+
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

// setDirection sets the flowchart's direction, reading Mermaid's arrows for it.
// It is Pele's setDirection.
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

// setClass gives the nodes, edges and subgraphs of ids, separated by commas, a class.
// It is Pele's setClass.
func (d *DB) setClass(ids, className string) {
	for _, id := range strings.Split(ids, ",") {
		if n := d.nodes[id]; n != nil {
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
// It is Pele's setTooltip.
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
// It is Pele's setLink.
func (d *DB) setLink(ids, link, target string) {
	u := chars.Trim(link)
	for _, id := range strings.Split(ids, ",") {
		if v := d.nodes[id]; v != nil {
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
// It is Pele's setClickEvent.
func (d *DB) setClickEvent(ids string) { d.setClass(ids, "clickable") }

// setAccTitle sets the accessible title, without the white space it starts with.
// It is Pele's setAccTitle.
func (d *DB) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, without the white space each of its
// lines but the first starts with: text.replace(/\n\s+/g, '\n').
// It is Pele's setAccDescription.
func (d *DB) setAccDescription(text string) {
	d.AccDescr = chars.DropIndents(text)
}

// docItem is what a statement of a subgraph's body gives it, Pele's DocItem: a string, a list
// of node ids, or a direction.
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

// hasSpace reports whether s has white space in it, the /\s/.test of Pele's addSubGraph.
func hasSpace(s string) bool { return strings.ContainsFunc(s, chars.IsSpaceRune) }

// addSubGraph adds a subgraph of what its body gave, or adds to the one of the same id, as
// Pele's addSubGraph does, and gives its id. A title that is the id itself, with white space
// in it, is a title only.
func (d *DB) addSubGraph(idText *Text, list []docItem, titleText *Text) string {
	id := ""
	hasID := idText != nil
	if hasID {
		id = chars.Trim(idText.Text)
	}
	title := ""
	if titleText != nil {
		title = titleText.Text
	}
	if idText != nil && idText == titleText && hasSpace(title) {
		hasID = false
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

	lt := LabelMarkdown
	if titleText != nil {
		lt = labelType(string(titleText.Type))
	}
	sg := &Subgraph{ID: id, Title: chars.Trim(title), Classes: []string{}, Dir: dir, LabelType: lt, Nodes: []string{}}
	// A node belongs to the first subgraph that lists it, and a subgraph never contains itself.
	for _, nodeID := range nodeList {
		if !d.members[nodeID] && nodeID != id {
			sg.Nodes = append(sg.Nodes, nodeID)
		}
	}
	for _, nodeID := range sg.Nodes {
		d.members[nodeID] = true
	}

	if existing := d.subgraphLookup[id]; existing != nil {
		existing.Nodes = append(existing.Nodes, sg.Nodes...)
	} else {
		d.Subgraphs = append(d.Subgraphs, sg)
		d.subgraphLookup[id] = sg
	}
	return id
}

// destructLink reads an arrow, as Pele's destructLink does: end is the arrow, or its end where
// start is the start of an arrow with a label between. An arrow whose start and end do not
// agree is of type and stroke "INVALID".
func (d *DB) destructLink(end, start string) LinkInfo {
	info := destructEndLink(end)
	if start == "" {
		return info
	}
	startInfo := destructStartLink(start)
	if startInfo.Stroke != info.Stroke {
		return LinkInfo{Type: "INVALID", Stroke: "INVALID"}
	}
	if startInfo.Type == "arrow_open" {
		startInfo.Type = info.Type
	} else {
		if startInfo.Type != info.Type {
			return LinkInfo{Type: "INVALID", Stroke: "INVALID"}
		}
		startInfo.Type = "double_" + startInfo.Type
	}
	if startInfo.Type == "double_arrow" {
		startInfo.Type = "double_arrow_point"
	}
	startInfo.Length = info.Length
	return startInfo
}
