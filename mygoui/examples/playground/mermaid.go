package main

import (
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

// mermaid is the editor's language for Mermaid's text: its comments, which the editor
// toggles with Cmd+/, its brackets and quotes, and the tokenizer that highlights it.
var mermaid = &editor.Language{
	Name:        "Mermaid",
	Extensions:  []string{".mmd", ".mermaid"},
	LineComment: "%%",
	Brackets:    []editor.BracketPair{{Open: "(", Close: ")"}, {Open: "[", Close: "]"}, {Open: "{", Close: "}"}},
	AutoClose: []editor.BracketPair{
		{Open: "(", Close: ")"}, {Open: "[", Close: "]"}, {Open: "{", Close: "}"},
		{Open: `"`, Close: `"`, NotIn: []editor.TokenKind{editor.TokenString}},
	},
	TabSize:   4,
	Tokenizer: editor.TokenizerFunc(tokenizeMermaid),
}

// The states a line of Mermaid's text can end in, as bits. The text starts in 0.
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
)

// keywords are words the diagrams have in common, or that many of them use, highlighted
// wherever they are not part of a label. It is a short list on purpose: the highlighting
// is to help read the text, and a word left plain does no harm.
var keywords = map[string]bool{
	"subgraph": true, "end": true, "direction": true, "click": true, "style": true,
	"classDef": true, "class": true, "linkStyle": true, "call": true, "href": true,
	"participant": true, "actor": true, "as": true, "loop": true, "alt": true, "else": true,
	"opt": true, "par": true, "and": true, "critical": true, "break": true, "rect": true,
	"note": true, "Note": true, "over": true, "activate": true, "deactivate": true,
	"autonumber": true, "create": true, "destroy": true, "box": true, "title": true,
	"section": true, "state": true, "accTitle": true, "accDescr": true, "dateFormat": true,
	"axisFormat": true, "tickInterval": true, "excludes": true, "includes": true,
	"todayMarker": true, "commit": true, "branch": true, "checkout": true, "merge": true,
	"cherry-pick": true, "x-axis": true, "y-axis": true, "bar": true, "line": true,
	"columns": true, "group": true, "service": true, "junction": true, "showData": true,
	"axis": true, "curve": true, "set": true, "union": true, "component": true,
}

// directions are the ways a diagram runs, highlighted as constants.
var directions = map[string]bool{"TB": true, "TD": true, "BT": true, "RL": true, "LR": true}

// tokenizeMermaid highlights a line of Mermaid's text: front matter and directives as meta,
// comments, strings, numbers, colors, arrows and the other operators, the word that says
// the diagram's type on its first line, and common keywords. Inside brackets, which hold a
// node's label, and after a colon, which starts a message or a value, words are plain
// text, so that a label that says "end" does not look like a keyword.
func tokenizeMermaid(line string, state uint64, toks []editor.Token) ([]editor.Token, uint64) {
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
	state |= declared
	depth := 0     // of the brackets around the scan
	label := false // past a colon that starts a label or a value
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
			add(j, editor.TokenNumber)
			i = j
		case isWordStart(c):
			j := wordEnd(line, i)
			word := line[i:j]
			kind := editor.TokenText
			switch {
			case declaration:
				kind = editor.TokenKeyword
				declaration = false
			case depth > 0 || label:
			case directions[word]:
				kind = editor.TokenConstant
			case keywords[word]:
				kind = editor.TokenKeyword
			}
			add(j, kind)
			i = j
		case isArrow(c):
			j := i
			for j < len(line) && isArrow(line[j]) {
				j++
			}
			kind := editor.TokenText
			if s := line[i:j]; len(s) > 1 || s == ">" || s == "<" {
				kind = editor.TokenOperator
			}
			add(j, kind)
			i = j
		case c == ':':
			j := i
			for j < len(line) && line[j] == ':' {
				j++
			}
			if j-i == 1 && depth == 0 {
				label = true
			}
			add(j, editor.TokenPunctuation)
			i = j
		case strings.IndexByte("([{", c) >= 0:
			depth++
			add(i+1, editor.TokenPunctuation)
			i++
		case strings.IndexByte(")]}", c) >= 0:
			depth = max(0, depth-1)
			add(i+1, editor.TokenPunctuation)
			i++
		case strings.IndexByte("|;,@", c) >= 0:
			add(i+1, editor.TokenPunctuation)
			i++
		default:
			add(i+1, editor.TokenText)
			i++
		}
	}
	return toks, state
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
