package settings

import (
	"fmt"
	"regexp"
	"strings"
)

// Key addresses one setting. It is one or more segments joined by "/", which
// namespaces settings the way directories do — "agent/model", "web/theme".
//
// The grammar is narrow on purpose. Keys become paths in [FS], so anything
// that could escape the root, name a hidden file, or mean something to a
// filesystem is rejected outright rather than sanitized: no empty segments,
// no "." or "..", no leading dot, nothing outside the allowed characters.
type Key string

// segment is one path component: alphanumerics, underscore and hyphen,
// optionally dotted (agent.model), never starting or ending with a dot.
var segment = regexp.MustCompile(`^[A-Za-z0-9_-]+(\.[A-Za-z0-9_-]+)*$`)

// Validate reports whether the key is well formed, wrapping [ErrInvalidKey]
// with the reason. It is a pure check: no store is consulted, so callers can
// validate before they connect to anything.
func (k Key) Validate() error {
	if k == "" {
		return fmt.Errorf("%w: empty", ErrInvalidKey)
	}
	if len(k) > maxKeyLen {
		return fmt.Errorf("%w: longer than %d bytes", ErrInvalidKey, maxKeyLen)
	}
	for _, s := range strings.Split(string(k), "/") {
		if !segment.MatchString(s) {
			return fmt.Errorf("%w: bad segment %q in %q", ErrInvalidKey, s, string(k))
		}
	}
	return nil
}

// maxKeyLen keeps a key comfortably inside the path limits of every
// filesystem an adapter might sit on.
const maxKeyLen = 512
