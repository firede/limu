package class

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// A Member is an attribute or a method of a class, as Pele's ClassMember is.
type Member struct {
	ID         string
	MemberType string // "method" or "attribute"
	Visibility string // "#", "+", "~", "-" or ""
	Classifier string // "$" for static, "*" for abstract, or ""
	Parameters string
	ReturnType string
}

func isVisibility(s string) bool { return s == "#" || s == "+" || s == "~" || s == "-" }

// NewMember reads a member as Pele's ClassMember does. Mermaid throws a TypeError for a method
// with no opening parenthesis, such as `run)`; it is kept here as an attribute, so it is
// shown as written.
func NewMember(input, memberType string) *Member {
	m := &Member{MemberType: memberType}
	if memberType != "method" || !m.parseMethod(input) {
		m.parseAttribute(input)
	}
	if strings.HasPrefix(m.ID, " ") {
		m.ID = " " + chars.Trim(m.ID)
	} else {
		m.ID = chars.Trim(m.ID)
	}
	return m
}

// lineEnd is where the line that starts at byte i of s ends, and the bytes of what ends it.
func lineEnd(s string, i int) (end, size int) {
	for i < len(s) {
		c, n := chars.UnitAt(s, i)
		if chars.IsLineEnd(c) {
			return i, n
		}
		i += n
	}
	return len(s), 1
}

// parseMethod splits a method as Mermaid's /([#+~-])?(.+)\((.*)\)([\s$*])?(.*)([$*])?/ does,
// found in one pass, as Pele's parseMethod does: the first line with an opening parenthesis
// after its first character and a closing one later.
func (m *Member) parseMethod(input string) bool {
	n := len(input)
	for start := 0; start < n; {
		end, size := lineEnd(input, start)
		closing := end - 1
		for closing > start && input[closing] != ')' {
			closing--
		}
		open := closing - 1
		for open > start && input[open] != '(' {
			open--
		}
		if open > start {
			first := input[start : start+1]
			visible := isVisibility(first) && open > start+1
			from := start
			if visible {
				m.Visibility = first
				from++
			}
			m.ID = input[from:open]
			m.Parameters = chars.Trim(input[open+1 : closing])
			rest := closing + 1
			classifier := ""
			if rest < n {
				c, size := chars.UnitAt(input, rest)
				if c == '$' || c == '*' {
					classifier = input[rest : rest+1]
					rest++
				} else if chars.IsSpace(c) {
					rest += size
				}
			}
			stop, _ := lineEnd(input, rest)
			m.ReturnType = chars.Trim(input[rest:stop])
			if classifier == "" && m.ReturnType != "" {
				if last := m.ReturnType[len(m.ReturnType)-1]; last == '$' || last == '*' {
					classifier = string(last)
					m.ReturnType = m.ReturnType[:len(m.ReturnType)-1]
				}
			}
			m.Classifier = classifier
			return true
		}
		start = end + size
	}
	m.MemberType = "attribute"
	return false
}

func (m *Member) parseAttribute(input string) {
	if input == "" {
		return
	}
	if first := input[:1]; isVisibility(first) {
		m.Visibility = first
	}
	if last := input[len(input)-1]; last == '$' || last == '*' {
		m.Classifier = string(last)
	}
	from, to := 0, len(input)
	if m.Visibility != "" {
		from = 1
	}
	if m.Classifier != "" {
		to--
	}
	m.ID = input[from:to]
}

// DisplayText is the member as a class shows it, as Pele's getDisplayDetails writes it:
// generic types in angle brackets, a method with its parameters and return type.
func (m *Member) DisplayText() string {
	text := m.Visibility + common.ParseGenericTypes(m.ID)
	if m.MemberType == "method" {
		text += "(" + common.ParseGenericTypes(chars.Trim(m.Parameters)) + ")"
		if m.ReturnType != "" {
			text += " : " + common.ParseGenericTypes(m.ReturnType)
		}
	}
	return chars.Trim(text)
}
