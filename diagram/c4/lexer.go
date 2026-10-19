package c4

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// tokens are the tokens of a text: the text of token i is src[starts[i]:ends[i]], as Pele's
// Tokens hold it. Positions are bytes, where Pele's are code units; they mark the same text.
type tokens struct {
	types, starts, ends []int
}

// The start conditions of the lexer, Pele's S.
const (
	sInitial = iota
	sStruct
	sAttribute
	sString
	sStringKV
	sStringKVKey
	sStringKVValue
	sAcc
	sAccDescrMultiline
)

var keywords = map[string]int{
	"C4Context":           tC4Context,
	"C4Container":         tC4Container,
	"C4Component":         tC4Component,
	"C4Dynamic":           tC4Dynamic,
	"C4Deployment":        tC4Deployment,
	"Person":              tPerson,
	"Person_Ext":          tPersonExt,
	"System":              tSystem,
	"SystemDb":            tSystemDB,
	"SystemQueue":         tSystemQueue,
	"System_Ext":          tSystemExt,
	"SystemDb_Ext":        tSystemExtDB,
	"SystemQueue_Ext":     tSystemExtQueue,
	"Container":           tContainer,
	"ContainerDb":         tContainerDB,
	"ContainerQueue":      tContainerQueue,
	"Container_Ext":       tContainerExt,
	"ContainerDb_Ext":     tContainerExtDB,
	"ContainerQueue_Ext":  tContainerExtQueue,
	"Component":           tComponent,
	"ComponentDb":         tComponentDB,
	"ComponentQueue":      tComponentQueue,
	"Component_Ext":       tComponentExt,
	"ComponentDb_Ext":     tComponentExtDB,
	"ComponentQueue_Ext":  tComponentExtQueue,
	"Enterprise_Boundary": tEnterpriseBoundary,
	"System_Boundary":     tSystemBoundary,
	"Boundary":            tBoundary,
	"Container_Boundary":  tContainerBoundary,
	"Deployment_Node":     tNode,
	"Node":                tNode,
	"Node_L":              tNodeL,
	"Node_R":              tNodeR,
	"Rel":                 tRel,
	"BiRel":               tBiRel,
	"Rel_Up":              tRelU,
	"Rel_U":               tRelU,
	"Rel_Down":            tRelD,
	"Rel_D":               tRelD,
	"Rel_Left":            tRelL,
	"Rel_L":               tRelL,
	"Rel_Right":           tRelR,
	"Rel_R":               tRelR,
	"Rel_Back":            tRelB,
	"RelIndex":            tRelIndex,
	"UpdateElementStyle":  tUpdateElStyle,
	"UpdateRelStyle":      tUpdateRelStyle,
	"UpdateLayoutConfig":  tUpdateLayoutConfig,
}

// directions are the words after `direction` in Pele's RE_DIRECTION, in its order.
var directions = [...]string{"TB", "BT", "RL", "LR"}

// lineEnd is where the line that byte p is on ends: the first line terminator at or after p,
// as RE_LINE_END finds it, or the end of src.
func lineEnd(src string, p int) int {
	for q := p; q < len(src); {
		c, size := chars.UnitAt(src, q)
		if chars.IsLineEnd(c) {
			return q
		}
		q += size
	}
	return len(src)
}

// matchDirection is the end of what /.*direction\s+WORD[^\n]*/y matches at p, or -1. The greedy
// .* backtracks from the end of the line, so the last `direction` on the line that is followed
// by white space and the word is the one matched; \s+ takes all the white space after it, line
// breaks too, as nothing else lets the word follow.
func matchDirection(src string, p, end int, word string) int {
	const keyword = "direction"
	for d := end - len(keyword); d >= p; d-- {
		if !strings.HasPrefix(src[d:], keyword) {
			continue
		}
		q := d + len(keyword)
		ws := chars.SpaceRun(src, q)
		if ws == q || !strings.HasPrefix(src[ws:], word) {
			continue
		}
		q = ws + len(word)
		if k := strings.IndexByte(src[q:], '\n'); k >= 0 {
			return q + k
		}
		return len(src)
	}
	return -1
}

// tokenize reads a text into tokens as Pele's tokenize does, which follows the lexer Jison
// compiles from c4Diagram.jison: its rule order, its start conditions, and what it returns once
// the input runs out in each of them.
func tokenize(src string) tokens {
	n := len(src)
	var t tokens
	state := sInitial
	accValue := tAccTitleValue
	p := 0
	done := false
	// Where the next `direction` is, -2 until the first search, and the end of the line it was
	// last looked for on, as Pele keeps them so as not to search the whole text again for each
	// statement.
	dirNext := -2
	lineAt := -1
	dirFailed := 0

	emit := func(typ, s, e int) {
		t.types = append(t.types, typ)
		t.starts = append(t.starts, s)
		t.ends = append(t.ends, e)
		p = e
	}
	// at is the code unit at q, -1 past the end, as charCodeAt's NaN.
	at := func(q int) int { return chars.Unit(src, q) }
	upTo := func(ch byte, from int) int {
		if from > n {
			return n
		}
		if q := strings.IndexByte(src[from:], ch); q >= 0 {
			return from + q
		}
		return n
	}
	// statement matches word\s[^#\n;]+.
	statement := func(word string, typ int) bool {
		if !strings.HasPrefix(src[p:], word) {
			return false
		}
		q := p + len(word)
		c, size := chars.UnitAt(src, q)
		if !chars.IsSpace(c) {
			return false
		}
		q += size
		from := q
		for q < n {
			if c := src[q]; c == '#' || c == '\n' || c == ';' {
				break
			}
			q++
		}
		if q == from {
			return false
		}
		emit(typ, p, q)
		return true
	}

	for {
		// Jison notes the end of input one call before it reports it, so a rule that matches
		// nothing gets one last turn.
		if done {
			emit(tEnd, n, n)
			break
		}
		if p >= n {
			done = true
		}
		c := at(p)

		switch state {
		case sInitial:
			if p >= n {
				emit(tEOF, n, n)
				continue
			}
			if dirNext != -1 && p >= dirFailed {
				if dirNext < p {
					if k := strings.Index(src[p:], "direction"); k >= 0 {
						dirNext = p + k
					} else {
						dirNext = -1
					}
				}
				if dirNext != -1 {
					if lineAt < p {
						lineAt = lineEnd(src, p)
					}
					if dirNext < lineAt {
						matched := false
						for i := 0; i < 4 && !matched; i++ {
							if e := matchDirection(src, p, lineAt, directions[i]); e >= 0 {
								emit(tDirectionTB+i, p, e)
								matched = true
							}
						}
						if matched {
							continue
						}
						// No direction statement starts anywhere in the rest of this line either.
						dirFailed = lineAt
					}
				}
			}
			if c == 't' && statement("title", tTitle) {
				continue
			}
			if c == 'a' {
				if statement("accDescription", tAccDescription) {
					continue
				}
				title := strings.HasPrefix(src[p:], "accTitle")
				if title || strings.HasPrefix(src[p:], "accDescr") {
					q := chars.SpaceRun(src, p+8)
					d := at(q)
					if d == ':' {
						typ := tAccDescr
						accValue = tAccDescrValue
						if title {
							typ = tAccTitle
							accValue = tAccTitleValue
						}
						emit(typ, p, chars.SpaceRun(src, q+1))
						state = sAcc
						continue
					}
					if d == '{' && !title {
						p = chars.SpaceRun(src, q+1)
						state = sAccDescrMultiline
						continue
					}
				}
			}
			if c == '%' && at(p+1) == '%' {
				// A comment takes the line breaks after it, so no NEWLINE follows.
				q := upTo('\n', p+2)
				for q < n && (src[q] == '\n' || src[q] == '\r') {
					q++
				}
				p = q
				continue
			}
			if chars.IsSpace(c) {
				q := p
				last := -1
				for q < n {
					d, size := chars.UnitAt(src, q)
					if !chars.IsSpace(d) {
						break
					}
					if d == '\n' {
						last = q
					}
					q += size
				}
				if last != -1 {
					emit(tNewline, p, last+1)
				} else {
					p = q
				}
				continue
			}
			if c == '{' {
				emit(tLBrace, p, p+1)
				continue
			}
			if c == '}' {
				emit(tRBrace, p, p+1)
				continue
			}
			if chars.IsWord(c) {
				q := p + 1
				for q < n && chars.IsWord(int(src[q])) {
					q++
				}
				if q-p <= 19 {
					if typ, ok := keywords[src[p:q]]; ok {
						if typ > tC4Deployment {
							state = sStruct
						}
						emit(typ, p, q)
						continue
					}
				}
			}
			emit(tError, p, p)
			return t

		case sStruct:
			switch {
			case p >= n:
				emit(tEOFInStruct, n, n)
			case c == '(':
				q := p + 1
				for at(q) == ' ' {
					q++
				}
				if at(q) == ',' {
					emit(tAttributeEmpty, p, q+1)
				} else {
					p++
				}
				state = sAttribute
			case c == ')':
				p++
				state = sInitial
			default:
				emit(tError, p, p)
				return t
			}

		case sAttribute:
			if p >= n {
				continue
			}
			switch c {
			case ')':
				p++
				state = sInitial
			case ',':
				if at(p+1) == ',' {
					emit(tAttributeEmpty, p, p+2)
				} else {
					p++
				}
			default:
				q := p
				for at(q) == ' ' {
					q++
				}
				switch at(q) {
				case '"':
					if at(q+1) == '"' {
						emit(tAttributeEmpty, p, q+2)
					} else {
						p = q + 1
						state = sString
					}
				case '$':
					p = q + 1
					state = sStringKV
				default:
					// An unquoted value runs to the next comma, through closing parentheses and
					// line breaks.
					emit(tStr, p, upTo(',', p))
				}
			}

		case sString:
			if c == '"' {
				p++
				state = sAttribute
			} else {
				emit(tStr, p, upTo('"', p))
			}

		case sStringKV:
			emit(tStrKey, p, upTo('=', p))
			state = sStringKVKey

		case sStringKVKey:
			if p >= n {
				continue
			}
			q := p + 1
			if c == '=' {
				for at(q) == ' ' {
					q++
				}
			}
			if c != '=' || at(q) != '"' {
				emit(tError, p, p)
				return t
			}
			p = q + 1
			state = sStringKVValue

		case sStringKVValue:
			if p >= n {
				continue
			}
			if c == '"' {
				p++
				state = sAttribute
			} else {
				emit(tStrValue, p, upTo('"', p))
			}

		case sAcc:
			emit(accValue, p, upTo('\n', p))
			state = sInitial

		default:
			if c == '}' {
				p++
				state = sInitial
			} else {
				emit(tAccDescrMultilineValue, p, upTo('}', p))
			}
		}
	}
	return t
}
