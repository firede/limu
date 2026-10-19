// Package packet parses Mermaid's packet diagrams and lays them out, as Pele's
// src/diagrams/packet does: a tokenizer and a parser that read what Mermaid's Langium grammar
// reads, buildPacket, which splits the fields into rows, and Layout, which lays the rows out
// as a scene.
//
// Bits are numbers as JavaScript's are, float64, as a field may be written wider than any
// integer holds.
package packet

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Block is a field, or the piece of one that falls in a row, Pele's PacketBlock.
type Block struct {
	Start, End float64
	// Bits is as Mermaid counts it: one less than the width for the pieces of a field split
	// across rows.
	Bits  float64
	Label string
}

// Model is a parsed packet diagram, Pele's PacketModel. A string Pele leaves undefined is
// empty.
type Model struct {
	Title, AccTitle, AccDescr string
	BitsPerRow                float64
	Rows                      [][]Block
}

// maxRows is the most rows a packet has, as Mermaid stops at.
const maxRows = 10000

// packetOption is a number of the packet's config if it is at least min, at most max, or
// fallback, as Pele's packetOption reads it.
func packetOption(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["packet"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo {
		return min(v, hi)
	}
	return fallback
}

func invalid(message string) *diagram.Error {
	return diagram.NewError(message, diagram.Semantic, "packet")
}

// buildPacket makes the model of a packet's tree, as Pele's buildPacket does: each field
// starts where the one before it ends, and is split where the rows end.
func buildPacket(tree *ast, config diagram.Config, title string) (*Model, error) {
	num := chars.NumberString
	bitsPerRow := math.Floor(packetOption(config, "bitsPerRow", 32, 1, 4096))
	var rows [][]Block
	lastBit := -1.0
	var word []Block
	row := 1.0

	for _, block := range tree.blocks {
		start, end, bits := block.start, block.end, block.bits
		if block.hasStart && block.hasEnd && end < start {
			return nil, invalid("Packet block " + num(start) + " - " + num(end) + " is invalid. End must be greater than start.")
		}
		if !block.hasStart {
			start = lastBit + 1
		}
		if start != lastBit+1 {
			shown := start
			if block.hasEnd {
				shown = end
			}
			return nil, invalid("Packet block " + num(start) + " - " + num(shown) + " is not contiguous. It should start from " + num(lastBit+1) + ".")
		}
		if block.hasBits && bits == 0 {
			return nil, invalid("Packet block " + num(start) + " is invalid. Cannot have a zero bit field.")
		}
		if !block.hasEnd {
			n := 1.0
			if block.hasBits {
				n = bits
			}
			end = start + n - 1
		}
		if !block.hasBits {
			bits = end - start + 1
		}
		lastBit = end

		for float64(len(word)) <= bitsPerRow+1 && len(rows) < maxRows {
			rowEnd := float64(row*bitsPerRow) - 1
			if end <= rowEnd {
				word = append(word, Block{start, end, bits, block.label})
				if end == rowEnd {
					rows = append(rows, word)
					word = nil
					row++
				}
				break
			}
			word = append(word, Block{start, rowEnd, rowEnd - start, block.label})
			rows = append(rows, word)
			word = nil
			row++
			start = rowEnd + 1
			bits = end - start
		}
	}
	if len(word) > 0 {
		rows = append(rows, word)
	}

	m := &Model{Title: tree.title, BitsPerRow: bitsPerRow, Rows: rows}
	if m.Title == "" {
		m.Title = title
	}
	if tree.accTitle != "" {
		m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.accDescr != "" {
		m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	return m, nil
}

// Parse parses a packet diagram's text, as the parse of Pele's packet diagram does: source is
// what diagram.Prepared.Source gives, config its configuration, and title the title its front
// matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string) (*Model, error) {
	tree, err := parsePacket(source)
	if err != nil {
		return nil, err
	}
	return buildPacket(tree, config, title)
}

// ParsePrepared parses a prepared packet diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Config, p.Title)
}
