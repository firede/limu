package packet

import (
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// The kinds of token, by index in packetTokens, as Pele's T, and those of the hidden types
// after them.
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
	tWhitespace
	tYAML
)

// packetTokens is Pele's PACKET_TOKENS.
var packetTokens = []common.TokenType{
	{Name: "packet-beta", Literal: "packet-beta"},
	common.Keyword("packet"),
	// A front matter block also starts with dashes, so Langium lets it win over a dash.
	{Name: "-", Literal: "-", Longer: []int{tYAML}},
	{Name: "+", Literal: "+"},
	{Name: ":", Literal: ":"},
	common.AccDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	common.IntToken,
	common.StringToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
}

var packetTable = common.Prepare(packetTokens)

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
	toks, err := common.Tokenize(src, packetTable, "packet")
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
	r := &common.Reader{Tokens: toks, Types: packetTokens, Diagram: "packet"}
	tree = &ast{}

	endOfLine := func() {
		if r.Kind() == -1 {
			return
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		for r.Kind() == tNewline {
			r.I++
		}
	}

	for r.Kind() == tNewline {
		r.I++
	}
	if !r.Accept(tPacketBeta) {
		r.Expect(tPacket)
	}
	for r.Kind() != -1 {
		switch r.Kind() {
		case tNewline:
			r.I++
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
			endOfLine()
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
			endOfLine()
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
			endOfLine()
		case tInt, tPlus:
			var block blockAst
			if r.Accept(tPlus) {
				block.bits, block.hasBits = number(r.Expect(tInt)), true
			} else {
				block.start, block.hasStart = number(r.Take()), true
				if r.Accept(tDash) {
					block.end, block.hasEnd = number(r.Expect(tInt)), true
				}
			}
			r.Expect(tColon)
			block.label = common.StringValue(r.Expect(tString))
			tree.blocks = append(tree.blocks, block)
			endOfLine()
		default:
			r.Fail("a field, a title, or a line break")
		}
	}
	return tree, nil
}
