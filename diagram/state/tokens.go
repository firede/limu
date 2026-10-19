package state

// token is a kind of token of the state diagram lexer, named and ordered as Pele's T in
// src/diagrams/state/tokens.ts, with t_ for T.
type token uint8

const (
	t_END token = iota
	t_ERROR
	t_NL
	t_SD
	t_DESCR
	t_ARROW
	t_HIDE_EMPTY
	t_scale
	t_WIDTH
	t_COMPOSIT_STATE
	t_STRUCT_START
	t_STRUCT_STOP
	t_STATE_DESCR
	t_AS
	t_ID
	t_FORK
	t_JOIN
	t_CHOICE
	t_CONCURRENT
	t_note
	t_NOTE_TEXT
	t_acc_title
	t_acc_title_value
	t_acc_descr
	t_acc_descr_value
	t_acc_descr_multiline_value
	t_CLICK
	t_STRING
	t_HREF
	t_classDef
	t_CLASSDEF_ID
	t_DEFAULT_CLASSDEF_ID
	t_CLASSDEF_STYLEOPTS
	t_DEFAULT
	t_style
	t_STYLE_IDS
	t_STYLEDEF_STYLEOPTS
	t_class
	t_CLASSENTITY_IDS
	t_STYLECLASS
	t_direction_tb
	t_direction_bt
	t_direction_rl
	t_direction_lr
	t_EDGE_STATE
	t_STYLE_SEPARATOR
	t_left_of
	t_right_of
	t_INVALID
)

// tokenNames are the names the grammar gives the tokens, as Pele's TOKEN_NAMES, for the
// messages of syntax errors.
var tokenNames = [...]string{
	"$end",
	"INVALID",
	"NL",
	"SD",
	"DESCR",
	"-->",
	"HIDE_EMPTY",
	"scale",
	"WIDTH",
	"COMPOSIT_STATE",
	"STRUCT_START",
	"STRUCT_STOP",
	"STATE_DESCR",
	"AS",
	"ID",
	"FORK",
	"JOIN",
	"CHOICE",
	"CONCURRENT",
	"note",
	"NOTE_TEXT",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"CLICK",
	"STRING",
	"HREF",
	"classDef",
	"CLASSDEF_ID",
	"DEFAULT_CLASSDEF_ID",
	"CLASSDEF_STYLEOPTS",
	"DEFAULT",
	"style",
	"STYLE_IDS",
	"STYLEDEF_STYLEOPTS",
	"class",
	"CLASSENTITY_IDS",
	"STYLECLASS",
	"direction_tb",
	"direction_bt",
	"direction_rl",
	"direction_lr",
	"EDGE_STATE",
	"STYLE_SEPARATOR",
	"left_of",
	"right_of",
	"INVALID",
}
