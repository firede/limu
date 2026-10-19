package url

import (
	"encoding/json"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them.

func TestSanitizeMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/url.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		URL []struct {
			Input     string  `json:"input"`
			Sanitized string  `json:"sanitized"`
			Safe      string  `json:"safe"`
			Link      *string `json:"link"`
			Image     *string `json:"image"`
			Mail      *string `json:"mail"`
		} `json:"url"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
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
