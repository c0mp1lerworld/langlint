// Package pseudonymizer derives one-way pseudonyms for analytics identifiers
// (A8). It lives in shared/ because both the api and the provisioner entry
// points materialize and purge analytics data, and they must agree on the
// pseudonym of a given identifier.
package pseudonymizer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Pseudonymizer applies a keyed HMAC-SHA256 to an identifier, turning it into a
// stable pseudonym that cannot be reversed without the secret (A8).
type Pseudonymizer struct {
	key []byte
}

// New builds a pseudonymizer from APP_PSEUDONYM_SECRET. An empty secret is
// rejected: without a key the pseudonym would be a plain digest.
func New(secret string) (*Pseudonymizer, error) {
	if secret == "" {
		return nil, fmt.Errorf("pseudonym secret must not be empty")
	}
	return &Pseudonymizer{key: []byte(secret)}, nil
}

// Pseudonymize returns the hex-encoded HMAC-SHA256 of id under the configured
// secret. It is deterministic: the same id and secret always yield the same
// pseudonym, so analytics can be keyed and queried consistently.
func (p *Pseudonymizer) Pseudonymize(id string) string {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil))
}
