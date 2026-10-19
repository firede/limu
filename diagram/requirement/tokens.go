package requirement

// token is a kind of token of the requirement lexer, named and ordered as Pele's T in
// src/diagrams/requirement/tokens.ts, with t_ for T.
type token uint8

const (
	t_END token = iota
	t_ERROR
	t_title
	t_acc_title
	t_acc_title_value
	t_acc_descr
	t_acc_descr_value
	t_acc_descr_multiline_value
	t_direction_tb
	t_direction_bt
	t_direction_rl
	t_direction_lr
	t_NEWLINE
	t_EOF
	t_RD
	t_STRUCT_START
	t_STRUCT_STOP
	t_STYLE_SEPARATOR
	t_COLONSEP
	t_ID
	t_TEXT
	t_RISK
	t_VERIFYMTHD
	t_REQUIREMENT
	t_FUNCTIONAL_REQUIREMENT
	t_INTERFACE_REQUIREMENT
	t_PERFORMANCE_REQUIREMENT
	t_PHYSICAL_REQUIREMENT
	t_DESIGN_CONSTRAINT
	t_LOW_RISK
	t_MED_RISK
	t_HIGH_RISK
	t_VERIFY_ANALYSIS
	t_VERIFY_DEMONSTRATION
	t_VERIFY_INSPECTION
	t_VERIFY_TEST
	t_ELEMENT
	t_CONTAINS
	t_COPIES
	t_DERIVES
	t_SATISFIES
	t_VERIFIES
	t_REFINES
	t_TRACES
	t_TYPE
	t_DOCREF
	t_STYLE
	t_CLASSDEF
	t_CLASS
	t_ALPHA
	t_COLON
	t_SEMICOLON
	t_PERCENT
	t_MINUS
	t_BRKT
	t_END_ARROW_L
	t_END_ARROW_R
	t_LINE
	t_qString
	t_unqString
	t_COMMA
)

// tokenNames are the names Jison's messages give the tokens, as Pele's TOKEN_NAMES.
var tokenNames = [...]string{
	"$end",
	"INVALID",
	"title",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"direction_tb",
	"direction_bt",
	"direction_rl",
	"direction_lr",
	"NEWLINE",
	"EOF",
	"RD",
	"STRUCT_START",
	"STRUCT_STOP",
	"STYLE_SEPARATOR",
	"COLONSEP",
	"ID",
	"TEXT",
	"RISK",
	"VERIFYMTHD",
	"REQUIREMENT",
	"FUNCTIONAL_REQUIREMENT",
	"INTERFACE_REQUIREMENT",
	"PERFORMANCE_REQUIREMENT",
	"PHYSICAL_REQUIREMENT",
	"DESIGN_CONSTRAINT",
	"LOW_RISK",
	"MED_RISK",
	"HIGH_RISK",
	"VERIFY_ANALYSIS",
	"VERIFY_DEMONSTRATION",
	"VERIFY_INSPECTION",
	"VERIFY_TEST",
	"ELEMENT",
	"CONTAINS",
	"COPIES",
	"DERIVES",
	"SATISFIES",
	"VERIFIES",
	"REFINES",
	"TRACES",
	"TYPE",
	"DOCREF",
	"STYLE",
	"CLASSDEF",
	"CLASS",
	"ALPHA",
	"COLON",
	"SEMICOLON",
	"PERCENT",
	"MINUS",
	"BRKT",
	"END_ARROW_L",
	"END_ARROW_R",
	"LINE",
	"qString",
	"unqString",
	"COMMA",
}

// keywords are the words the lexer reads as keywords, lowered, as Pele's KEYWORDS. Every
// keyword is matched without regard to case and must end at a word boundary.
var keywords = map[string]token{
	"requirementdiagram":     t_RD,
	"id":                     t_ID,
	"text":                   t_TEXT,
	"risk":                   t_RISK,
	"verifymethod":           t_VERIFYMTHD,
	"requirement":            t_REQUIREMENT,
	"functionalrequirement":  t_FUNCTIONAL_REQUIREMENT,
	"interfacerequirement":   t_INTERFACE_REQUIREMENT,
	"performancerequirement": t_PERFORMANCE_REQUIREMENT,
	"physicalrequirement":    t_PHYSICAL_REQUIREMENT,
	"designconstraint":       t_DESIGN_CONSTRAINT,
	"low":                    t_LOW_RISK,
	"medium":                 t_MED_RISK,
	"high":                   t_HIGH_RISK,
	"analysis":               t_VERIFY_ANALYSIS,
	"demonstration":          t_VERIFY_DEMONSTRATION,
	"inspection":             t_VERIFY_INSPECTION,
	"test":                   t_VERIFY_TEST,
	"element":                t_ELEMENT,
	"contains":               t_CONTAINS,
	"copies":                 t_COPIES,
	"derives":                t_DERIVES,
	"satisfies":              t_SATISFIES,
	"verifies":               t_VERIFIES,
	"refines":                t_REFINES,
	"traces":                 t_TRACES,
	"type":                   t_TYPE,
	"docref":                 t_DOCREF,
	"style":                  t_STYLE,
	"classdef":               t_CLASSDEF,
	"class":                  t_CLASS,
}
