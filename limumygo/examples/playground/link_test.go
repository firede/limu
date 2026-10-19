package main

import (
	"strings"
	"testing"
)

// A link holds the text in its fragment, and gives it back as it was.
func TestShareLink(t *testing.T) {
	for _, src := range []string{
		"flowchart LR\n  A[Start] --> B{Is it?}\n",
		"pie\n  \"a & b\": 1\n  \"c + d = 100%\": 2\n  \"#hash\": 3\n",
		"flowchart TD\n  A[用户打开购物车] --> B([在庫切れ]) --> C[(결제 완료)]\n",
		"",
	} {
		link, ok := shareLink(siteURL, src)
		if !ok || !strings.HasPrefix(link, siteURL+"#source=") {
			t.Errorf("shareLink(%q) = %q, %v", src, link, ok)
			continue
		}
		_, hash, _ := strings.Cut(link, siteURL)
		if got, ok := linkedSource(hash); !ok || got != src {
			t.Errorf("linkedSource(%q) = %q, %v; want %q", hash, got, ok, src)
		}
	}
	// A text whose link would pass maxLinkHash has none.
	if _, ok := shareLink(siteURL, strings.Repeat("&", maxLinkHash/3)); ok {
		t.Error("a link too long to share")
	}
}

// The playground reads the fragments the website writes, as URLSearchParams encodes them,
// and none that has no text.
func TestLinkedSource(t *testing.T) {
	cases := []struct {
		hash, src string
		ok        bool
	}{
		// URLSearchParams({source: "flowchart LR\n  A --> B~*"}).toString()
		{"#source=flowchart+LR%0A++A+--%3E+B%7E*", "flowchart LR\n  A --> B~*", true},
		{"#theme=dark&source=pie", "pie", true},
		{"#source=", "", true},
		{"#source=a%zz&other=1", "", false},
		{"#other=1", "", false},
		{"#", "", false},
		{"", "", false},
		{"source=pie", "", false},
	}
	for _, tc := range cases {
		if src, ok := linkedSource(tc.hash); src != tc.src || ok != tc.ok {
			t.Errorf("linkedSource(%q) = %q, %v; want %q, %v", tc.hash, src, ok, tc.src, tc.ok)
		}
	}
}
