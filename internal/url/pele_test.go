package url

import (
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them.

func TestSanitizeMatchesPele(t *testing.T) {
	var p struct {
		URL []struct {
			Input     string  `json:"input"`
			Sanitized string  `json:"sanitized"`
			Safe      string  `json:"safe"`
			Link      *string `json:"link"`
			Image     *string `json:"image"`
			Mail      *string `json:"mail"`
			// Internal and InternalMail are what internalLink says of the input as a note's
			// name, with the schemes of Link and of Mail.
			Internal     bool `json:"internal"`
			InternalMail bool `json:"internalMail"`
		} `json:"url"`
	}
	if err := fixture.Load("testdata/pele/url.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.URL) < 1000 {
		t.Fatalf("only %d cases", len(p.URL))
	}
	opt := func(s string, ok bool) *string {
		if !ok {
			return nil
		}
		return &s
	}
	same := func(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
	show := func(s *string) string {
		if s == nil {
			return "undefined"
		}
		return *s
	}
	failed := 0
	for _, c := range p.URL {
		got := SanitizeURL(c.Input)
		safe := SafeURL(c.Input)
		link := opt(LinkURL(got, true, nil))
		image := opt(ImageURL(got, true, nil))
		mail := opt(LinkURL(got, true, []string{"mailto", "obsidian"}))
		if got != c.Sanitized || safe != c.Safe || !same(link, c.Link) || !same(image, c.Image) || !same(mail, c.Mail) {
			t.Errorf("%q\n got %q, safe %q, link %q, image %q, mail %q\nPele %q, safe %q, link %q, image %q, mail %q", c.Input,
				got, safe, show(link), show(image), show(mail), c.Sanitized, c.Safe, show(c.Link), show(c.Image), show(c.Mail))
			if failed++; failed > 20 {
				t.Fatal("too many failures")
			}
		}
		internal := InternalLink(c.Input, true, nil)
		internalMail := InternalLink(c.Input, true, []string{"mailto", "obsidian"})
		if internal != c.Internal || internalMail != c.InternalMail {
			t.Errorf("%q as a note's name: internal %v, with mail %v; Pele %v, %v", c.Input, internal, internalMail, c.Internal, c.InternalMail)
			if failed++; failed > 20 {
				t.Fatal("too many failures")
			}
		}
	}
}

func TestLinksOff(t *testing.T) {
	if _, ok := LinkURL("https://x", false, nil); ok {
		t.Error("links off")
	}
	if _, ok := ImageURL("https://x", false, nil); ok {
		t.Error("images off")
	}
}

// The cases of Pele's tests of internalLink, tests/util/url.test.ts.
func TestInternalLink(t *testing.T) {
	for _, name := range []string{"Note", `Folder\note`, "Note%23heading", "👩‍💻", "Note#heading", "  Note  "} {
		if !InternalLink(name, true, nil) {
			t.Errorf("%q refused", name)
		}
	}
	for _, name := range []string{"javascript:alert(1)", "java￾script:alert(1)", `\\nevil.example`, "\t/\\nevil.example", "/\t/evil.example"} {
		if InternalLink(name, true, []string{"about"}) {
			t.Errorf("%q allowed", name)
		}
	}
}
