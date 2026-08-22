package ids_test

import (
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/ids"
)

func TestMint(t *testing.T) {
	cases := map[string]struct{ name, fallback, want string }{
		"words":          {"In Review", "x", "in-review-n"},
		"punctuation":    {"Won't Fix!", "x", "won-t-fix-n"},
		"already a stem": {"acme-site", "project", "acme-site-n"},
		"nothing usable": {"日本語", "project", "project-n"},
		"very long": {
			strings.Repeat("a very long name ", 5), "x",
			"a-very-long-name-a-very-long-name-a-very-n",
		},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			if got := ids.Mint(c.name, c.fallback, "n"); got != c.want {
				t.Errorf("Mint(%q) = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// The fallback is what keeps an id saying what kind of thing it is when the
// name reduces to nothing.
func TestMintFallsBackPerCaller(t *testing.T) {
	if got := ids.Mint("!!!", "project", "n"); got != "project-n" {
		t.Errorf("Mint = %q, want the caller's fallback", got)
	}
	if got := ids.Mint("!!!", "x", "n"); got != "x-n" {
		t.Errorf("Mint = %q, want the caller's fallback", got)
	}
}

// An id is used as a path segment and a URL segment, so whatever a name throws
// at it, the result has to be safe as both.
func TestMintedIDsAreSafeSegments(t *testing.T) {
	for _, name := range []string{"In Review", "../escape", "a/b", "日本語", "Won't Fix!", ""} {
		id := ids.Mint(name, "x", "4f7k")
		if strings.ContainsAny(id, `/\ .:?#%`) {
			t.Errorf("Mint(%q) = %q, which is not safe as a segment", name, id)
		}
		if err := ids.Validate("thing", id); err != nil {
			t.Errorf("Mint(%q) = %q, which does not validate: %v", name, id, err)
		}
	}
}

func TestValidateRefusesWhatItCouldNotHaveMinted(t *testing.T) {
	for name, id := range map[string]string{
		"empty":           "",
		"capitals":        "Bug",
		"underscore":      "bug_9c2x",
		"space":           "in review",
		"dot":             "low.4j6k",
		"trailing hyphen": "bug-",
		"leading hyphen":  "-bug",
		"path separator":  "bug/9c2x",
		"too long":        strings.Repeat("a", 200),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ids.Validate("thing", id); err == nil {
				t.Errorf("Validate accepted %q", id)
			}
		})
	}
}

// Two ids minted from one name must differ, or the nonce is not doing its job.
func TestNonceIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 500)
	for range 500 {
		n := ids.Nonce()
		if _, dup := seen[n]; dup {
			t.Fatalf("Nonce repeated %q within 500 draws", n)
		}
		seen[n] = struct{}{}
	}
}
