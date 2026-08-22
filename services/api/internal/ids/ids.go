// Package ids mints the readable identifiers this system addresses things by.
//
// An id is a primary key a person can read: a stem taken from the name at
// creation, plus a suffix that makes it unique (ADR-0009). The stem is frozen
// and will go stale — rename a thing and its id keeps the old words — and that
// is the point, because a stem that followed the name would be a name.
//
// One package rather than one per domain: projects and the tracker mint ids
// the same way, and two copies of this would drift into two grammars, which
// is exactly the kind of duplication ADR-0001 exists to prevent.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Shape is what a minted id looks like. Deliberately narrow, so an id is
// always safe as both a path segment and a URL segment.
var Shape = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// MaxLen keeps an id inside the path limits of any filesystem it lands on.
const MaxLen = 128

// maxStemLen bounds the readable half, so a long name still yields a
// manageable id.
const maxStemLen = 40

var notStem = regexp.MustCompile(`[^a-z0-9]+`)

// Mint builds an id from a name and a nonce.
//
// A name with nothing usable in it — punctuation, or a script this reduction
// does not handle — still gets an id, because the nonce alone addresses it and
// refusing the name would be worse. Callers supply the fallback stem so an id
// still says what kind of thing it is.
func Mint(name, fallback, nonce string) string {
	stem := stemOf(name)
	if stem == "" {
		stem = fallback
	}
	return stem + "-" + nonce
}

// Nonce is the unique half of an id: short, lowercase, unguessable.
func Nonce() string {
	var b [3]byte
	// crypto/rand.Read is documented never to fail as of Go 1.24.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Validate reports whether a string could be an id this system minted. It
// checks the shape only: an id that looks minted may still name nothing.
func Validate(kind, id string) error {
	switch {
	case id == "":
		return fmt.Errorf("empty %s id", kind)
	case len(id) > MaxLen:
		return fmt.Errorf("%s id longer than %d bytes", kind, MaxLen)
	case !Shape.MatchString(id):
		return fmt.Errorf("%s id %q is not one this system mints", kind, id)
	default:
		return nil
	}
}

func stemOf(name string) string {
	stem := notStem.ReplaceAllString(strings.ToLower(name), "-")
	stem = strings.Trim(stem, "-")
	if len(stem) > maxStemLen {
		stem = strings.Trim(stem[:maxStemLen], "-")
	}
	return stem
}
