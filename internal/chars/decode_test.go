package chars

import "testing"

// What Node's TextDecoder makes of the same bytes.
func TestDecode(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"plain", "plain"},
		{"a\xffb", "a�b"},
		{"\xe2\x82", "�"},
		{"\xe2\x82x", "�x"},
		{"\xed\xa0\xbd", "���"},
		{"\xf0\x9f\x98", "�"},
		{"\xf0\x9f\x98\x80", "😀"},
		{"\xc0\xaf", "��"},
		{"\xf4\x90\x80\x80", "����"},
		{"\xe0\x80\x80", "���"},
		{"\x80\x80", "��"},
		{"\xf5", "�"},
		{"é\xe2\x82é", "é�é"},
	} {
		if got := Decode(c.in); got != c.want {
			t.Errorf("Decode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
