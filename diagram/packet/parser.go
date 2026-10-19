package packet

import (
	"strconv"

	"github.com/firede/limu/diagram"
)

// The kinds of token, by index in packetTokens, as Pele's T.
const (
	tPacketBeta = iota
	tPacket
	tDash
	tPlus
	tColon
	tAccDescr
	tAccTitle
	tTitle
	tInt
	tString
	tNewline
)

// intToken is the table's INT, /0|[1-9][0-9]*(?!\.)/y: a lone 0, or digits not followed by a
// dot, which gives back the last digit rather than fail when they are.
var intToken = tokenType{name: "INT", anyFirst: true, match: func(s string, at int) int {
	switch {
	case s[at] == '0':
		return at + 1
	case s[at] < '1' || s[at] > '9':
		return -1
	}
	end := at + 1
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end < len(s) && s[end] == '.' {
		if end-1 > at {
			return end - 1
		}
		return -1
	}
	return end
}}

// packetTokens is Pele's PACKET_TOKENS.
var packetTokens = []tokenType{
	{name: "packet-beta", literal: "packet-beta"},
	keyword("packet"),
	// A front matter block also starts with dashes, so Langium lets it win over a dash.
	{name: "-", literal: "-", longer: []int{12}},
	{name: "+", literal: "+"},
	{name: ":", literal: ":"},
	accDescrToken,
	accTitleToken,
	titleToken,
	intToken,
	stringToken,
	newlineToken,
	whitespaceToken,
	yamlToken,
	directiveToken,
	commentToken,
}

var packetTable = prepare(packetTokens)

// blockAst is Pele's PacketBlockAst; a number Pele leaves undefined has its Has field unset.
type blockAst struct {
	start, end, bits          float64
	hasStart, hasEnd, hasBits bool
	label                     string
}

// ast is Pele's PacketAst.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	blocks                             []blockAst
}

// number is JavaScript's Number() of an INT token's digits, which may be too many for a
// float64 to hold, and are then infinite.
func number(digits string) float64 {
	f, _ := strconv.ParseFloat(digits, 64)
	return f
}

// parsePacket parses a packet diagram's text into its tree, as Pele's parsePacket does.
func parsePacket(src string) (tree *ast, err error) {
	toks, err := tokenize(src, packetTokens, packetTable, "packet")
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			de, ok := e.(*diagram.Error)
			if !ok {
				panic(e)
			}
			tree, err = nil, de
		}
	}()
	r := &reader{toks: toks, types: packetTokens, typ: "packet"}
	tree = &ast{}

	endOfLine := func() {
		if r.kind() == -1 {
			return
		}
		if r.kind() != tNewline {
			r.fail("a line break or the end of input")
		}
		for r.kind() == tNewline {
			r.i++
		}
	}

	for r.kind() == tNewline {
		r.i++
	}
	if !r.accept(tPacketBeta) {
		r.expect(tPacket)
	}
	for r.kind() != -1 {
		switch r.kind() {
		case tNewline:
			r.i++
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = accDescrValue(r.take()), true
			endOfLine()
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = accTitleValue(r.take()), true
			endOfLine()
		case tTitle:
			tree.title, tree.hasTitle = titleValue(r.take()), true
			endOfLine()
		case tInt, tPlus:
			var block blockAst
			if r.accept(tPlus) {
				block.bits, block.hasBits = number(r.expect(tInt)), true
			} else {
				block.start, block.hasStart = number(r.take()), true
				if r.accept(tDash) {
					block.end, block.hasEnd = number(r.expect(tInt)), true
				}
			}
			r.expect(tColon)
			block.label = stringValue(r.expect(tString))
			tree.blocks = append(tree.blocks, block)
			endOfLine()
		default:
			r.fail("a field, a title, or a line break")
		}
	}
	return tree, nil
}
