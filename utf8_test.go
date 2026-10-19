package limu

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/firede/limu/internal/fixture"
)

// Pele reads a JavaScript string, which a browser decoded from UTF-8, and a string a browser
// decodes holds U+FFFD wherever its bytes were not UTF-8. A Go string can hold such bytes,
// and a lone surrogate, which a JavaScript string can hold and UTF-8 cannot, is written in
// them as the three bytes of its WTF-8 form. So the contract is that bytes that are not
// UTF-8 are laid out as the U+FFFD a browser's decoder reads them as, and the text package's
// fixture checks that U+FFFD is measured as Pele measures it.
//
// For a few sources of every diagram type, from the cases the render suites drew, the test
// puts bytes that are not UTF-8 at places in the source and expects the same scene, or the
// same error, as with the U+FFFD that stand for them, and a scene of valid UTF-8 only.
func TestInvalidUTF8IsReplacementCharacter(t *testing.T) {
	paths, err := filepath.Glob("diagram/*/testdata/pele/render.json.gz")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no render fixtures: %v", err)
	}
	// Each is bytes that are not UTF-8 and what a browser's decoder makes of them.
	bad := []struct{ bytes, decoded string }{
		{"\xff", "\uFFFD"},
		{"\xed\xa0\xbd", "\uFFFD\uFFFD\uFFFD"}, // a lone high surrogate, as WTF-8 writes it
		{"\xe4\xb8", "\uFFFD"},                 // a sequence that is cut short
	}
	for _, path := range paths {
		data, err := fixture.Read(strings.TrimSuffix(path, ".gz"))
		if err != nil {
			t.Fatal(err)
		}
		var f struct {
			Cases []struct {
				Src string `json:"src"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(data, &f); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for i, c := range f.Cases {
			if i == 20 {
				break
			}
			// A place on the first line after its first word, and one at each third of the
			// rest, each where a character starts.
			places := []int{strings.IndexAny(c.Src, " \n") + 1, len(c.Src) / 3, 2 * len(c.Src) / 3}
			for k, at := range places {
				for at > 0 && at < len(c.Src) && !utf8.RuneStart(c.Src[at]) {
					at++
				}
				if at <= 0 {
					continue
				}
				b := bad[(i+k)%len(bad)]
				src, want := c.Src[:at]+b.bytes+c.Src[at:], c.Src[:at]+b.decoded+c.Src[at:]
				gotScene, gotErr := Layout(src, nil)
				wantScene, wantErr := Layout(want, nil)
				if errString(gotErr) != errString(wantErr) || !reflect.DeepEqual(gotScene, wantScene) {
					t.Errorf("%s case %d: %q laid out unlike %q: %v, %v", path, i, src, want, gotErr, wantErr)
					continue
				}
				if gotScene != nil {
					if s := invalidText(reflect.ValueOf(gotScene)); s != "" {
						t.Errorf("%s case %d: the scene holds %q, which is not UTF-8", path, i, s)
					}
				}
			}
		}
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// invalidText is the first string in v, or what it points to or holds, that is not UTF-8, or
// "".
func invalidText(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return v.String()
		}
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return invalidText(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if s := invalidText(v.Index(i)); s != "" {
				return s
			}
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			if s := invalidText(it.Key()); s != "" {
				return s
			}
			if s := invalidText(it.Value()); s != "" {
				return s
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if s := invalidText(v.Field(i)); s != "" {
				return s
			}
		}
	}
	return ""
}
