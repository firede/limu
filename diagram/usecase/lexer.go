package usecase

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// tokens are a source's tokens: the type of each, and the bytes it starts and ends at. Three
// tEOF tokens end them, so that the parser may look three tokens ahead anywhere.
type tokens struct {
	types  []int
	starts []int
	ends   []int
}

const (
	classWord        = 1
	classPunctuation = 2
)

// class says of each ASCII character whether it is part of a word, or punctuation of CSS:
// Pele's CLASS.
var class = func() (t [128]uint8) {
	for c := '0'; c <= '9'; c++ {
		t[c] = classWord
	}
	for c := 'A'; c <= 'Z'; c++ {
		t[c] = classWord
		t[c+32] = classWord
	}
	t['_'] = classWord
	for _, c := range "!#$&*+/=?^|~" {
		t[c] = classPunctuation
	}
	return
}()

// keywordLength is the length of each keyword, by its type of token: Pele's KEYWORD_LENGTH.
var keywordLength = [...]int{0, 12, 5, 14, 3, 9, 2, 2, 2, 2, 2, 4, 3, 4, 8, 5, 5, 7, 6, 4, 5}

// at is the byte of s at i, or -1 past its end, as charCodeAt is for ASCII. It stands for the
// src.charCodeAt calls of Pele's lexer and parser.
func at(s string, i int) int {
	if i < 0 || i >= len(s) {
		return -1
	}
	return int(s[i])
}

// isWord reports whether character c is part of a word, as Pele's isWord does.
func isWord(c int) bool { return c >= 0 && c < 128 && class[c]&classWord != 0 }

// startsWithNoCase reports whether word, in lower case, starts at byte i of s in any case:
// Mermaid's INCLUDE and EXTEND are the only keywords matched without regard to case. It is
// Pele's startsWithNoCase.
func startsWithNoCase(s string, i int, word string) bool {
	for k := 0; k < len(word); k++ {
		if at(s, i+k)|32 != int(word[k]) {
			return false
		}
	}
	return true
}

// keywordAt is the keyword that is a prefix of the text at i, tried in the order of Mermaid's
// token list, as Pele's keywordAt finds it.
func keywordAt(s string, i, c int) int {
	has := func(word string) bool { return strings.HasPrefix(s[i:], word) }
	switch c {
	case 'a':
		if has("actor") {
			return tACTOR
		}
	case 's':
		if has("systemBoundary") {
			return tSYSTEM_BOUNDARY
		}
		if has("style") {
			return tSTYLE
		}
	case 'e':
		if has("end") {
			return tEND
		}
		if startsWithNoCase(s, i, "extend") {
			return tEXTEND
		}
	case 'E':
		if startsWithNoCase(s, i, "extend") {
			return tEXTEND
		}
	case 'd':
		if has("direction") {
			return tDIRECTION
		}
	case 'T':
		if has("TD") {
			return tTD
		}
		if has("TB") {
			return tTB
		}
	case 'B':
		if has("BT") {
			return tBT
		}
	case 'L':
		if has("LR") {
			return tLR
		}
	case 'R':
		if has("RL") {
			return tRL
		}
	case 'n':
		if has("note") {
			return tNOTE
		}
	case 'f':
		if has("for") {
			return tFOR
		}
		if has("false") {
			return tFALSE
		}
	case 'j':
		if has("json") {
			return tJSON
		}
	case 'c':
		if has("classDef") {
			return tCLASS_DEF
		}
		if has("class") {
			return tCLASS
		}
	case 'i', 'I':
		if startsWithNoCase(s, i, "include") {
			return tINCLUDE
		}
	case 't':
		if has("true") {
			return tTRUE
		}
	case 'u':
		if has("usecase-beta") {
			return tUSECASE
		}
	}
	return 0
}

// lineColumn is the line and column of byte offset, as Pele's lineColumn counts them: a
// carriage return and a line feed after it end one line, and the column is in UTF-16 code
// units.
func lineColumn(s string, offset int) (line, column int) {
	line = 1
	lineStart := 0
	for i := 0; i < offset; i++ {
		c := s[i]
		if c != '\n' && c != '\r' {
			continue
		}
		if c == '\r' && i+1 < offset && s[i+1] == '\n' {
			i++
		}
		line++
		lineStart = i + 1
	}
	return line, chars.Len16(s[lineStart:offset]) + 1
}

// unit is byte offset i of s in UTF-16 code units, as Pele's offsets count.
func unit(s string, i int) int { return chars.Len16(s[:i]) }

// locationText is where the bytes from start to end are, as Pele's locationText writes it.
func locationText(s string, start, end int) string {
	line, column := lineColumn(s, start)
	return "line " + strconv.Itoa(line) + ", column " + strconv.Itoa(column) +
		" [" + strconv.Itoa(unit(s, start)) + "," + strconv.Itoa(unit(s, end)) + ")"
}

// stringEnd is the end of a quoted string that opens at i, or -1 when the line ends first, as
// Pele's stringEnd finds it.
func stringEnd(s string, i int, quote byte) int {
	for k := i + 1; k < len(s); k++ {
		c := s[k]
		if c == quote {
			return k + 1
		}
		if c == '\n' || c == '\r' {
			return -1
		}
	}
	return -1
}

// startsToken reports whether a token starts at i: anything but a quote that opens no string,
// as Pele's startsToken does.
func startsToken(s string, i int) bool {
	c := s[i]
	if c != '"' && c != '\'' {
		return true
	}
	return c == '"' && at(s, i+1) == '`' || stringEnd(s, i, c) != -1
}

// lexError is Mermaid's error for the first character no token starts with, from byte start
// to end, the characters its lexer skipped before one did, as Pele's lexError words it.
func lexError(s string, start, end int) *diagram.Error {
	line, column := lineColumn(s, start)
	offset, length := unit(s, start), chars.Len16(s[start:end])
	return &diagram.Error{
		Code: diagram.Syntax,
		Message: "Error lexing usecase diagram: unexpected character: ->" + s[start:start+1] + "<- at offset: " +
			strconv.Itoa(offset) + ", skipped " + strconv.Itoa(length) + " characters. at line " + strconv.Itoa(line) +
			", column " + strconv.Itoa(column) + " [" + strconv.Itoa(offset) + "," + strconv.Itoa(offset+length) + ")",
		Type: "usecase", Line: line, Column: column,
	}
}

// tokenize gives the tokens of Mermaid's Chevrotain lexer for use case diagrams, as Pele's
// tokenize does: the first pattern in its list that matches wins, a keyword gives way to a
// longer identifier, and `json id@` and `<<` switch to modes that read a JSON object and a
// stereotype. Every character it looks for is ASCII, so it steps through bytes; a run of
// other characters is one LABEL_SYMBOL.
func tokenize(s string) (tokens, error) {
	var t tokens
	push := func(typ, start, end int) {
		t.types = append(t.types, typ)
		t.starts = append(t.starts, start)
		t.ends = append(t.ends, end)
	}
	n := len(s)
	i := 0
	// Whether only blanks came before on this line, where comments and accessibility lines
	// may start.
	lineStart := true
	noBrace := false

	for i < n {
		c := int(s[i])
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		var typ int
		end := i + 1
		fresh := false

		switch {
		case isWord(c):
			typ = 0
			if c == 'a' && lineStart && strings.HasPrefix(s[i:], "acc") {
				descr := strings.HasPrefix(s[i+3:], "Descr")
				if descr || strings.HasPrefix(s[i+3:], "Title") {
					j := i + 8
					for chars.IsBlank(at(s, j)) {
						j++
					}
					next := at(s, j)
					if next == ':' {
						if descr {
							typ = tACC_DESCR_LINE
						} else {
							typ = tACC_TITLE_LINE
						}
						end = j + 1
						for end < n && s[end] != '\n' && s[end] != '\r' {
							end++
						}
					} else if next == '{' && descr && !noBrace {
						if close := strings.IndexByte(s[j+1:], '}'); close == -1 {
							noBrace = true
						} else {
							typ = tACC_DESCR_BLOCK
							end = j + 1 + close + 1
						}
					}
				}
			} else if c == 'j' && strings.HasPrefix(s[i:], "json") && chars.IsBlank(at(s, i+4)) {
				j := i + 5
				for chars.IsBlank(at(s, j)) {
					j++
				}
				id := j
				for isWord(at(s, j)) {
					j++
				}
				if j > id {
					for chars.IsBlank(at(s, j)) {
						j++
					}
					if at(s, j) == '@' {
						j++
						for chars.IsBlank(at(s, j)) {
							j++
						}
						if at(s, j) == '{' {
							push(tJSON_DECLARATION_START, i, j)
							depth := 0
							quoted, escaped := false, false
							k := j
							for ; k < n; k++ {
								ch := s[k]
								if quoted {
									if escaped {
										escaped = false
									} else if ch == '\\' {
										escaped = true
									} else if ch == '"' {
										quoted = false
									}
								} else if ch == '"' {
									quoted = true
								} else if ch == '{' {
									depth++
								} else if ch == '}' {
									if depth--; depth == 0 {
										break
									}
								}
							}
							if k < n {
								push(tJSON_OBJECT_LITERAL, j, k+1)
								i = k + 1
							} else {
								push(tUNCLOSED_JSON_OBJECT_LITERAL, j, n)
								i = n
							}
							lineStart = false
							continue
						}
					}
				}
			}
			if typ == 0 {
				for end < n && isWord(int(s[end])) {
					end++
				}
				if keyword := keywordAt(s, i, c); keyword != 0 {
					// A keyword that is only the start of a longer word is an identifier.
					if keyword == tUSECASE || end-i == keywordLength[keyword] {
						typ = keyword
						end = i + keywordLength[keyword]
					} else {
						typ = tIDENTIFIER
					}
				} else if (c == 'o' || c == 'x') && at(s, i+1) == '-' && at(s, i+2) == '-' {
					if c == 'o' {
						typ = tBACKWARD_CIRCLE
					} else {
						typ = tBACKWARD_CROSS
					}
					end = i + 3
				} else if chars.IsDigit(c) {
					// `1mg` is an identifier, and `1.5px` is a number because the identifier
					// stops at the dot.
					digits := i + 1
					for chars.IsDigit(at(s, digits)) {
						digits++
					}
					if at(s, digits) == '.' && chars.IsDigit(at(s, digits+1)) {
						typ = tNUMBER
						end = digits + 2
						for chars.IsDigit(at(s, end)) {
							end++
						}
						for chars.IsLetter(at(s, end)) {
							end++
						}
					} else {
						typ = tIDENTIFIER
					}
				} else {
					typ = tIDENTIFIER
					for at(s, end) == '-' && isWord(at(s, end+1)) {
						typ = tCSS_IDENTIFIER
						end += 2
						for isWord(at(s, end)) {
							end++
						}
					}
				}
			}
		case c == '\n':
			typ = tNEWLINE
			fresh = true
		case c == '\r':
			typ = tNEWLINE
			fresh = true
			if at(s, end) == '\n' {
				end++
			}
		case c == '"' || c == '\'':
			if c == '"' && at(s, i+1) == '`' {
				if close := strings.Index(s[i+2:], "`\""); close == -1 {
					typ = tUNCLOSED_MARKDOWN_STRING
					end = n
				} else {
					typ = tMARKDOWN_STRING
					end = i + 2 + close + 2
				}
			} else {
				typ = tPLAIN_STRING
				end = stringEnd(s, i, byte(c))
				if end == -1 {
					resume := i + 1
					for resume < n && !startsToken(s, resume) {
						resume++
					}
					return t, lexError(s, i, resume)
				}
			}
		case c == '%':
			if lineStart && at(s, end) == '%' {
				typ = tCOMMENT
				for end < n && s[end] != '\n' && s[end] != '\r' {
					end++
				}
			} else {
				typ = tPERCENT
			}
		case c == '-':
			if at(s, end) != '-' {
				typ = tDASH
				break
			}
			for at(s, end) == '-' {
				end++
			}
			next := at(s, end)
			two := end-i == 2
			switch {
			case two && next == '|' && at(s, end+1) == '>':
				typ = tGENERALIZATION
				end += 2
			case next == '>':
				typ = tFORWARD_SOLID
				end++
			case two && (next == 'o' || next == 'x'):
				if next == 'o' {
					typ = tFORWARD_CIRCLE
				} else {
					typ = tFORWARD_CROSS
				}
				end++
			default:
				typ = tMARKERLESS_SOLID
			}
		case c == '.':
			next := at(s, end)
			switch {
			case next == '.' && at(s, end+1) == '>':
				typ = tDEPENDENCY_ARROW
				end += 2
			case chars.IsDigit(next):
				typ = tNUMBER
				for chars.IsDigit(at(s, end)) {
					end++
				}
				for chars.IsLetter(at(s, end)) {
					end++
				}
			default:
				typ = tDOT
			}
		case c == '<':
			if at(s, end) == '<' {
				push(tSTEREOTYPE_START, i, i+2)
				lineStart = false
				i += 2
				if i >= n {
					break
				}
				if at(s, i) == '>' && at(s, i+1) == '>' {
					typ = tSTEREOTYPE_END
					end = i + 2
					break
				}
				close := i
				found := false
				for ; close < n; close++ {
					ch := s[close]
					if ch == '\n' || ch == '\r' {
						break
					}
					if ch == '>' && at(s, close+1) == '>' {
						found = true
						break
					}
				}
				if found && chars.Trim(s[i:close]) != "" {
					push(tSTEREOTYPE_TEXT, i, close)
					typ = tSTEREOTYPE_END
					i = close
					end = close + 2
				} else {
					for close < n && s[close] != '\n' && s[close] != '\r' {
						close++
					}
					if close == i {
						for close < n && (s[close] == '\n' || s[close] == '\r') {
							close++
						}
						return t, lexError(s, i, close)
					}
					typ = tUNCLOSED_STEREOTYPE_TEXT
					end = close
				}
			} else if at(s, end) == '-' && at(s, end+1) == '-' {
				typ = tBACKWARD_SOLID
				end += 2
				for at(s, end) == '-' {
					end++
				}
			} else {
				typ = tLABEL_PUNCTUATION
			}
		case c == ':':
			if at(s, end) == ':' && at(s, end+1) == ':' {
				typ = tCLASS_SEPARATOR
				end += 2
			} else {
				typ = tCOLON
			}
		case c == '@':
			if at(s, end) == '{' {
				typ = tMETADATA_START
				end++
			} else {
				typ = tAT
			}
		case c == '{':
			typ = tLBRACE
		case c == '}':
			typ = tRBRACE
		case c == '[':
			typ = tLBRACKET
		case c == ']':
			typ = tRBRACKET
		case c == '(':
			typ = tLPAREN
		case c == ')':
			typ = tRPAREN
		case c == ',':
			typ = tCOMMA
		case c == '#':
			for chars.IsHex(at(s, end)) {
				end++
			}
			if end > i+1 {
				typ = tHASH_COLOR
			} else {
				typ = tCSS_PUNCTUATION
			}
		case c == '\\':
			if at(s, end) == ',' {
				typ = tCSS_ESCAPED_COMMA
				end++
			} else {
				typ = tLABEL_PUNCTUATION
			}
		case c < 127 && c > 32:
			if class[c]&classPunctuation != 0 {
				typ = tCSS_PUNCTUATION
			} else {
				typ = tLABEL_PUNCTUATION
			}
		default:
			typ = tLABEL_SYMBOL
			for end < n {
				ch := s[end]
				if ch >= 32 && ch < 127 || ch == '\t' || ch == '\n' || ch == '\r' {
					break
				}
				end++
			}
		}
		if i >= n {
			// `<<` at the very end of the text.
			break
		}

		push(typ, i, end)
		i = end
		lineStart = fresh
	}

	for k := 0; k < 3; k++ {
		push(tEOF, n, n)
	}
	return t, nil
}
