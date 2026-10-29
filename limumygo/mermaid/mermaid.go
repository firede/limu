// Package mermaid is Mermaid's text as a language of MyGo's code editor: its comments,
// brackets and quotes, and a tokenizer that highlights it.
//
//	buf := editor.NewBuffer(src)
//	buf.SetLanguage(mermaid.Language)
//
// The tokenizer reads the diagram's type from the line that declares it, as limu detects
// it, and highlights as keywords the words that start a statement of that type, as
// Pele's playground offers them for completion. Other words are names, and the words of a
// node's label or a message are a string, so that a label that says "end" does not look
// like a keyword. It sorts the text as the website's highlighter does, so that a diagram's
// text looks the same in the playground as in the website's code. It reads a line at a
// time and does not parse; the highlighting is to help read the text, and limu's errors
// are what say whether it is right.
package mermaid

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
)

// Language is Mermaid's text, for files named .mmd and .mermaid.
var Language = &editor.Language{
	Name:        "Mermaid",
	Extensions:  []string{".mmd", ".mermaid"},
	LineComment: "%%",
	Brackets:    []editor.BracketPair{{Open: "(", Close: ")"}, {Open: "[", Close: "]"}, {Open: "{", Close: "}"}},
	AutoClose: []editor.BracketPair{
		{Open: "(", Close: ")"}, {Open: "[", Close: "]"}, {Open: "{", Close: "}"},
		{Open: `"`, Close: `"`, NotIn: []editor.TokenKind{editor.TokenString}},
	},
	TabSize:   4,
	Tokenizer: editor.TokenizerFunc(Tokenize),
}

// The state a line ends in holds these bits, and from bit 8 the number of the diagram's
// type in limu.Types, plus one, once the line that declares it is read. The text starts
// in 0.
const (
	// inFront is inside the front matter, which a line of --- opens on the text's first line
	// and the next such line closes.
	inFront uint64 = 1 << iota
	// inDirective is inside a %%{…}%% directive that goes on past its line.
	inDirective
	// declared is past the line that says the diagram's type.
	declared
	// begun is past the text's first line, where front matter can no longer open.
	begun

	typeShift = 8
	typeMask  = 0xff << typeShift
)

// types are the types limu reads, in an order that does not change while the program
// runs, so that a line's state can name one by its number.
var types = limu.Types()

func typeOf(state uint64) diagram.Type {
	if n := int(state&typeMask>>typeShift) - 1; n >= 0 && n < len(types) {
		return types[n]
	}
	return ""
}

func withType(state uint64, t diagram.Type) uint64 {
	state &^= typeMask
	if n := slices.Index(types, t); n >= 0 {
		state |= uint64(n+1) << typeShift
	}
	return state
}

// Tokenize highlights a line of Mermaid's text, given the state the line before ended in, as
// the website's highlighter does (src/lib/highlight.ts of firede/limu-website): front matter
// and directives as meta, comments, the word that says the diagram's type, the keywords of its
// type, and directions; names as variables, and labels, in a node's shape, between the |s of
// an edge or after a colon, as strings; numbers, colors, and arrows and the other operators; a
// class given with ::: as a keyword, and the declarations of a style as properties and their
// values. It is the Tokenizer of Language.
func Tokenize(line string, state uint64, toks []editor.Token) ([]editor.Token, uint64) {
	first := state&begun == 0
	state |= begun
	trimmed := strings.TrimSpace(line)
	add := func(end int, k editor.TokenKind) {
		if n := len(toks); n > 0 && toks[n-1].Kind == k {
			toks[n-1].End = end
			return
		}
		toks = append(toks, editor.Token{End: end, Kind: k})
	}
	switch {
	case state&inFront != 0:
		if trimmed == "---" {
			state &^= inFront
		}
		add(len(line), editor.TokenMeta)
		return toks, state
	case first && strings.HasPrefix(line, "---") && trimmed == "---":
		add(len(line), editor.TokenMeta)
		return toks, state | inFront
	case state&inDirective != 0:
		if strings.Contains(line, "}%%") {
			state &^= inDirective
		}
		add(len(line), editor.TokenMeta)
		return toks, state
	case strings.HasPrefix(trimmed, "%%{"):
		if !strings.Contains(trimmed[3:], "}%%") {
			state |= inDirective
		}
		add(len(line), editor.TokenMeta)
		return toks, state
	case strings.HasPrefix(trimmed, "%%"):
		add(len(line), editor.TokenComment)
		return toks, state
	case trimmed == "":
		add(len(line), editor.TokenText)
		return toks, state
	}
	declaration := state&declared == 0
	if declaration {
		state |= declared
		if t, err := diagram.DetectType(trimmed); err == nil {
			state = withType(state, t)
		}
	}
	words := keywordsOf(typeOf(state))
	er := typeOf(state) == diagram.ER
	start := true     // no word of the statement has been read yet
	depth := 0        // of the brackets around the scan
	settings := 0     // the depth of the @{…} of a node's settings the scan is in, or 0
	label := false    // past a colon that starts a label or a value
	edge := false     // between the |s of an edge's label
	class := false    // past the ::: that gives a node a class
	styling := 0      // 1 past style, classDef or linkStyle, and 2 past what it styles
	declares := false // in the declarations of a style, as fill:#f9f,stroke-width:2px
	for i := 0; i < len(line); {
		c := line[i]
		switch {
		case c == '"':
			j := strings.IndexByte(line[i+1:], '"')
			if j < 0 {
				j = len(line)
			} else {
				j += i + 2
			}
			add(j, editor.TokenString)
			i = j
		case er && erLink(line[i:]) > 0:
			j := i + erLink(line[i:])
			add(j, editor.TokenOperator)
			start = false
			i = j
		case c == '#' && i+1 < len(line) && isHex(line[i+1]):
			j := i + 1
			for j < len(line) && isWord(line[j]) {
				j++
			}
			add(j, editor.TokenConstant)
			i = j
		case isDigit(c) && (i == 0 || !isWord(line[i-1])):
			j := i
			for j < len(line) && (isDigit(line[j]) || line[j] == '.' && j+1 < len(line) && isDigit(line[j+1])) {
				j++
			}
			if declares {
				// A length's unit, as in 2px, or a percentage.
				for j < len(line) && (isWordStart(line[j]) && line[j] < 0x80 || line[j] == '%') {
					j++
				}
			}
			add(j, editor.TokenNumber)
			start = false
			if styling == 1 {
				styling = 2
			}
			i = j
		case isWordStart(c):
			j := wordEnd(line, i)
			word := line[i:j]
			kind := editor.TokenVariable
			switch {
			case declaration:
				kind = editor.TokenKeyword
				declaration = false
			case declares:
				kind = editor.TokenText
				if colonAfter(line, j) {
					kind = editor.TokenProperty
				} else if j < len(line) && line[j] == '(' {
					kind = editor.TokenFunction
				}
			case styling == 2 && colonAfter(line, j):
				declares = true
				kind = editor.TokenProperty
			case class:
				kind = editor.TokenKeyword
			case settings > 0 && depth >= settings:
				switch {
				case colonAfter(line, j):
					kind = editor.TokenProperty
				case word == "true" || word == "false":
					kind = editor.TokenConstant
				default:
					kind = editor.TokenString
				}
			case depth > 0 || label || edge:
				kind = editor.TokenString
			case directions()[word]:
				kind = editor.TokenConstant
			case start && words.starts[word], words.inline[word]:
				kind = editor.TokenKeyword
				if start && (word == "style" || word == "classDef" || word == "linkStyle") {
					styling = 1
				}
			}
			if styling == 1 && kind != editor.TokenKeyword {
				styling = 2
			}
			add(j, kind)
			start, class = false, false
			i = j
		case isArrow(c):
			j := i
			for j < len(line) {
				switch {
				case isArrow(line[j]):
					j++
					continue
				case line[j] == '|' && (line[j-1] == '<' || j+1 < len(line) && line[j+1] == '>'):
					// The | of a class diagram's <|-- and --|>.
					j++
					continue
				}
				break
			}
			if s := line[i:j]; len(s) > 1 && (s[len(s)-1] == '-' || s[len(s)-1] == '=') &&
				j < len(line) && (line[j] == 'o' || line[j] == 'x') && (j+1 == len(line) || !isWord(line[j+1])) {
				// The circle or the cross a flowchart's link ends in, as in --o and --x.
				j++
			}
			kind := editor.TokenText
			if s := line[i:j]; len(s) > 1 || s == ">" || s == "<" {
				kind = editor.TokenOperator
			}
			add(j, kind)
			start = false
			i = j
		case c == ':':
			j := i
			for j < len(line) && line[j] == ':' {
				j++
			}
			switch {
			case j-i == 3:
				class = true
			case j-i == 1 && depth == 0 && !declares:
				label = true
			}
			add(j, editor.TokenPunctuation)
			i = j
		case strings.IndexByte("([{", c) >= 0:
			depth++
			if c == '{' && i > 0 && line[i-1] == '@' {
				settings = depth
			}
			add(i+1, editor.TokenPunctuation)
			start = false
			i++
		case strings.IndexByte(")]}", c) >= 0:
			if depth == settings {
				settings = 0
			}
			depth = max(0, depth-1)
			add(i+1, editor.TokenPunctuation)
			i++
		case c == ';' && depth == 0:
			// A semicolon ends a statement, and the next may start on the same line.
			add(i+1, editor.TokenPunctuation)
			start, label, edge, styling, declares = true, false, false, 0, false
			i++
		case c == '|':
			edge = !edge
			add(i+1, editor.TokenPunctuation)
			i++
		case strings.IndexByte(";,@", c) >= 0:
			add(i+1, editor.TokenPunctuation)
			i++
		default:
			add(i+1, editor.TokenText)
			i++
		}
	}
	return toks, state
}

// colonAfter reports whether a colon follows s[j:], after spaces: whether the word before j
// is the name of a property, as fill in fill:#f9f or shape in @{ shape: rect }.
func colonAfter(s string, j int) bool {
	for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
		j++
	}
	return j < len(s) && s[j] == ':' && (j+1 == len(s) || s[j+1] != ':')
}

// erLink is the length of the relationship an entity relationship diagram's line starts
// with, as ||--o{ or }|..|{: the cardinality of each end about a solid or dashed line, or 0.
func erLink(s string) int {
	if len(s) < 6 || !strings.Contains("|}", s[:1]) || !strings.Contains("|o", s[1:2]) ||
		s[2:4] != "--" && s[2:4] != ".." || !strings.Contains("o|", s[4:5]) || !strings.Contains("|{", s[5:6]) {
		return 0
	}
	return 6
}

// wordEnd is where the word at i ends: letters, digits and _, and a - between them, as in
// x-axis and stateDiagram-v2, but not the - of an arrow, as in A-->B.
func wordEnd(s string, i int) int {
	j := i
	for j < len(s) {
		switch {
		case isWord(s[j]):
			j++
		case s[j] == '-' && j+1 < len(s) && isWord(s[j+1]) && j > i:
			j++
		default:
			return j
		}
	}
	return j
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isHex(c byte) bool { return isDigit(c) || 'a' <= c|0x20 && c|0x20 <= 'f' }

// isWordStart reports whether c starts a word: a letter, _, or a byte of a character
// outside ASCII, which Mermaid takes as part of a name.
func isWordStart(c byte) bool { return 'a' <= c|0x20 && c|0x20 <= 'z' || c == '_' || c >= 0x80 }

func isWord(c byte) bool { return isWordStart(c) || isDigit(c) }

// isArrow reports whether c is part of an arrow or a link, as -->, ==>, -.->, ~~~ or ->>.
func isArrow(c byte) bool { return strings.IndexByte("-=.<>~*+&", c) >= 0 }
