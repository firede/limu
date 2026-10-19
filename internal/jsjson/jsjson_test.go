package jsjson

import (
	"testing"
	"unicode/utf16"
)

// The values and messages JSON.parse gives in Node, and what JSON.stringify writes of them.
func TestParse(t *testing.T) {
	for _, c := range []struct{ text, json, message string }{
		{`{"b":1,"a":[true,null,"x"],"2":0,"10":1,"1":2,"b":3}`, `{"1":2,"2":0,"10":1,"b":3,"a":[true,null,"x"]}`, ""},
		{` [1e400, -0.5, "é\n"] `, `[null,-0.5,"é\n"]`, ""},
		{`{"a":1,}`, "", `Expected double-quoted property name in JSON at position 7 (line 1 column 8)`},
		{`[1 2]`, "", `Expected ',' or ']' after array element in JSON at position 3 (line 1 column 4)`},
		{`tru`, "", `Unexpected end of JSON input`},
		{`01`, "", `Unexpected number in JSON at position 1 (line 1 column 2)`},
	} {
		v, message := Parse(utf16.Encode([]rune(c.text)))
		if message != c.message {
			t.Errorf("Parse(%q): message %q, want %q", c.text, message, c.message)
			continue
		}
		if message == "" {
			if got := Stringify(v); got != c.json {
				t.Errorf("Stringify(Parse(%q)) = %s, want %s", c.text, got, c.json)
			}
		}
	}
}
