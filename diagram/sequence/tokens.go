package sequence

// The tokens of the lexer, Pele's T, in its order.
const (
	tEND = iota
	tERROR
	tINVALID
	tNEWLINE
	tSD
	tCreate
	tBox
	tRestOfLine
	tEnd
	tAutonumber
	tNUM
	tOff
	tActivate
	tDeactivate
	tTitle
	tLegacyTitle
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tLoop
	tRect
	tOpt
	tAlt
	tPar
	tParOver
	tCritical
	tBreak
	tOption
	tAnd
	tElse
	tParticipant
	tAS
	tParticipantActor
	tDestroy
	tNote
	tOver
	tLinks
	tLink
	tProperties
	tDetails
	tCOMMA
	tLeftOf
	tRightOf
	tPLUS
	tMINUS
	tCENTRAL
	tACTOR
	tCONFIG_START
	tCONFIG_CONTENT
	tCONFIG_END
	tTXT
	tSOLID_OPEN_ARROW
	tDOTTED_OPEN_ARROW
	tSOLID_ARROW
	tBIDIRECTIONAL_SOLID_ARROW
	tDOTTED_ARROW
	tBIDIRECTIONAL_DOTTED_ARROW
	tSOLID_CROSS
	tDOTTED_CROSS
	tSOLID_POINT
	tDOTTED_POINT
	tSOLID_ARROW_TOP
	tSOLID_ARROW_BOTTOM
	tSTICK_ARROW_TOP
	tSTICK_ARROW_BOTTOM
	tSOLID_ARROW_TOP_DOTTED
	tSOLID_ARROW_BOTTOM_DOTTED
	tSTICK_ARROW_TOP_DOTTED
	tSTICK_ARROW_BOTTOM_DOTTED
	tSOLID_ARROW_TOP_REVERSE
	tSOLID_ARROW_BOTTOM_REVERSE
	tSTICK_ARROW_TOP_REVERSE
	tSTICK_ARROW_BOTTOM_REVERSE
	tSOLID_ARROW_TOP_REVERSE_DOTTED
	tSOLID_ARROW_BOTTOM_REVERSE_DOTTED
	tSTICK_ARROW_TOP_REVERSE_DOTTED
	tSTICK_ARROW_BOTTOM_REVERSE_DOTTED
	tokenCount
)

// tokenNames are the tokens as Mermaid's grammar spells them, for error messages, Pele's
// TOKEN_NAMES.
var tokenNames = [tokenCount]string{
	"$end", "INVALID", "INVALID", "NEWLINE", "SD", "create", "box", "restOfLine", "end", "autonumber", "NUM", "off",
	"activate", "deactivate", "title", "legacy_title", "acc_title", "acc_title_value", "acc_descr", "acc_descr_value",
	"acc_descr_multiline_value", "loop", "rect", "opt", "alt", "par", "par_over", "critical", "break", "option", "and",
	"else", "participant", "AS", "participant_actor", "destroy", "note", "over", "links", "link", "properties", "details",
	",", "left_of", "right_of", "+", "-", "()", "ACTOR", "CONFIG_START", "CONFIG_CONTENT", "CONFIG_END", "TXT",
	"SOLID_OPEN_ARROW", "DOTTED_OPEN_ARROW", "SOLID_ARROW", "BIDIRECTIONAL_SOLID_ARROW", "DOTTED_ARROW",
	"BIDIRECTIONAL_DOTTED_ARROW", "SOLID_CROSS", "DOTTED_CROSS", "SOLID_POINT", "DOTTED_POINT", "SOLID_ARROW_TOP",
	"SOLID_ARROW_BOTTOM", "STICK_ARROW_TOP", "STICK_ARROW_BOTTOM", "SOLID_ARROW_TOP_DOTTED", "SOLID_ARROW_BOTTOM_DOTTED",
	"STICK_ARROW_TOP_DOTTED", "STICK_ARROW_BOTTOM_DOTTED", "SOLID_ARROW_TOP_REVERSE", "SOLID_ARROW_BOTTOM_REVERSE",
	"STICK_ARROW_TOP_REVERSE", "STICK_ARROW_BOTTOM_REVERSE", "SOLID_ARROW_TOP_REVERSE_DOTTED",
	"SOLID_ARROW_BOTTOM_REVERSE_DOTTED", "STICK_ARROW_TOP_REVERSE_DOTTED", "STICK_ARROW_BOTTOM_REVERSE_DOTTED",
}
