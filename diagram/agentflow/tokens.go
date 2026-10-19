package agentflow

// token is a kind of token of the agent flow lexer, named and ordered as Pele's T in
// src/diagrams/agentflow/tokens.ts, with t_ for T.
type token uint8

const (
	t_END token = iota
	t_ERROR
	t_acc_title
	t_acc_title_value
	t_acc_descr
	t_acc_descr_value
	t_acc_descr_multiline_value
	t_SHAPE_DATA
	t_CALLBACKNAME
	t_CALLBACKARGS
	t_MD_STR
	t_STR
	t_STYLE
	t_DEFAULT
	t_LINKSTYLE
	t_INTERPOLATE
	t_CLASSDEF
	t_CLASS
	t_HREF
	t_CLICK
	t_GRAPH
	t_flow
	t_connector
	t_global
	t_end
	t_LINK_TARGET
	t_NODIR
	t_DIR
	t_direction_tb
	t_direction_bt
	t_direction_rl
	t_direction_lr
	t_direction_td
	t_LINK_ID
	t_NUM
	t_BRKT
	t_STYLE_SEPARATOR
	t_COLON
	t_AMP
	t_SEMI
	t_COMMA
	t_MULT
	t_LINK
	t_START_LINK
	t_EDGE_TEXT
	t_ELLIPSE_END
	t_TEXT
	t_ELLIPSE_START
	t_STADIUMEND
	t_STADIUMSTART
	t_SUBROUTINEEND
	t_SUBROUTINESTART
	t_VERTEX_WITH_PROPS_START
	t_TAGEND
	t_CYLINDEREND
	t_CYLINDERSTART
	t_DOUBLECIRCLEEND
	t_DOUBLECIRCLESTART
	t_TRAPEND
	t_INVTRAPEND
	t_TRAPSTART
	t_INVTRAPSTART
	t_TAGSTART
	t_UP
	t_SEP
	t_DOWN
	t_COMMENT
	t_NODE_STRING
	t_MINUS
	t_UNICODE_TEXT
	t_PIPE
	t_PE
	t_PS
	t_SQE
	t_SQS
	t_DIAMOND_STOP
	t_DIAMOND_START
	t_NEWLINE
	t_SPACE
	t_EOF
)

// tokenNames are the names Pele's parser gives the tokens in its errors, Pele's TOKEN_NAMES.
var tokenNames = [...]string{
	"$end", "INVALID", "acc_title", "acc_title_value", "acc_descr", "acc_descr_value",
	"acc_descr_multiline_value", "SHAPE_DATA", "CALLBACKNAME", "CALLBACKARGS", "MD_STR", "STR", "STYLE",
	"DEFAULT", "LINKSTYLE", "INTERPOLATE", "CLASSDEF", "CLASS", "HREF", "CLICK", "GRAPH", "flow",
	"connector", "global", "end", "LINK_TARGET", "NODIR", "DIR", "direction_tb", "direction_bt",
	"direction_rl", "direction_lr", "direction_td", "LINK_ID", "NUM", "BRKT", "STYLE_SEPARATOR", "COLON",
	"AMP", "SEMI", "COMMA", "MULT", "LINK", "START_LINK", "EDGE_TEXT", "-)", "TEXT", "(-", "STADIUMEND",
	"STADIUMSTART", "SUBROUTINEEND", "SUBROUTINESTART", "VERTEX_WITH_PROPS_START", "TAGEND", "CYLINDEREND",
	"CYLINDERSTART", "DOUBLECIRCLEEND", "DOUBLECIRCLESTART", "TRAPEND", "INVTRAPEND", "TRAPSTART",
	"INVTRAPSTART", "TAGSTART", "UP", "SEP", "DOWN", "COMMENT", "NODE_STRING", "MINUS", "UNICODE_TEXT",
	"PIPE", "PE", "PS", "SQE", "SQS", "DIAMOND_STOP", "DIAMOND_START", "NEWLINE", "SPACE", "EOF",
}
