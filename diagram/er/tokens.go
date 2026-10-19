package er

// token is a kind of token of the ER lexer, named and ordered as Pele's T in
// src/diagrams/er/tokens.ts, with t_ for T.
type token uint8

const (
	t_END token = iota
	t_ERROR
	t_ER_DIAGRAM
	t_EOF
	t_NEWLINE
	t_COLON
	t_STYLE_SEPARATOR
	t_BLOCK_START
	t_BLOCK_STOP
	t_SQS
	t_SQE
	t_acc_title
	t_acc_title_value
	t_acc_descr
	t_acc_descr_value
	t_acc_descr_multiline_value
	t_END_KW
	t_SUBGRAPH
	t_direction_tb
	t_direction_bt
	t_direction_rl
	t_direction_lr
	t_CLASSDEF
	t_UNICODE_TEXT
	t_STYLE_TEXT
	t_COMMA
	t_CLASS
	t_STYLE
	t_SEMI
	t_NUM
	t_BRKT
	t_ENTITY_NAME
	t_DECIMAL_NUM
	t_ENTITY_ONE
	t_ATTRIBUTE_WORD
	t_ATTRIBUTE_KEY
	t_COMMENT
	t_ZERO_OR_ONE
	t_ZERO_OR_MORE
	t_ONE_OR_MORE
	t_ONLY_ONE
	t_MD_PARENT
	t_NON_IDENTIFYING
	t_IDENTIFYING
	t_WORD
	// A character no rule names; the grammar gives it meaning only as `?` and `,` inside a
	// block.
	t_CHAR
)

// tokenNames are the names of the tokens as Jison's errors give them, Pele's TOKEN_NAMES.
var tokenNames = [...]string{
	"$end",
	"INVALID",
	"ER_DIAGRAM",
	"EOF",
	"NEWLINE",
	"COLON",
	"STYLE_SEPARATOR",
	"BLOCK_START",
	"BLOCK_STOP",
	"SQS",
	"SQE",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"END",
	"SUBGRAPH",
	"direction_tb",
	"direction_bt",
	"direction_rl",
	"direction_lr",
	"CLASSDEF",
	"UNICODE_TEXT",
	"STYLE_TEXT",
	"COMMA",
	"CLASS",
	"STYLE",
	"SEMI",
	"NUM",
	"BRKT",
	"ENTITY_NAME",
	"DECIMAL_NUM",
	"ENTITY_ONE",
	"ATTRIBUTE_WORD",
	"ATTRIBUTE_KEY",
	"COMMENT",
	"ZERO_OR_ONE",
	"ZERO_OR_MORE",
	"ONE_OR_MORE",
	"ONLY_ONE",
	"MD_PARENT",
	"NON_IDENTIFYING",
	"IDENTIFYING",
	"WORD",
	"CHAR",
}
