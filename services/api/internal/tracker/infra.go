package tracker

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// SystemClock reports the real time. Adapters take a [Clock] rather than
// calling time.Now so that tests can make time boring.
type SystemClock struct{}

// Now implements [Clock].
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// RandomIDs mints unguessable identifiers. Format is not part of any
// contract — nothing may parse an ID or infer anything from one.
type RandomIDs struct{}

// NewID implements [IDGenerator].
func (RandomIDs) NewID() string {
	var b [16]byte
	// crypto/rand.Read is documented never to fail as of Go 1.24.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
