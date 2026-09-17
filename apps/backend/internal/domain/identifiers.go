package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ID is an RFC 9562 UUID v7 generated with the standard library only (A7).
//
// The zero value is invalid: use NewID or MustNewID to obtain a valid one.
type ID [16]byte

// randRead is a seam for deterministic tests; it defaults to crypto/rand.Read.
var randRead = rand.Read

// NewID returns a new UUID v7 built from crypto/rand and the current time.
func NewID() (ID, error) {
	var id ID
	if _, err := randRead(id[:]); err != nil {
		return ID{}, fmt.Errorf("domain: generate id: %w", err)
	}

	ms := uint64(time.Now().UnixMilli())
	id[0] = byte(ms >> 40)
	id[1] = byte(ms >> 32)
	id[2] = byte(ms >> 24)
	id[3] = byte(ms >> 16)
	id[4] = byte(ms >> 8)
	id[5] = byte(ms)

	id[6] = (id[6] & 0x0f) | 0x70
	id[8] = (id[8] & 0x3f) | 0x80

	return id, nil
}

// MustNewID is like NewID but panics if the identifier cannot be generated.
func MustNewID() ID {
	id, err := NewID()
	if err != nil {
		panic(err)
	}
	return id
}

// ParseID parses the canonical string representation of a UUID.
func ParseID(s string) (ID, error) {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return ID{}, fmt.Errorf("domain: invalid id %q", s)
	}
	hexStr := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	raw, err := hex.DecodeString(hexStr)
	if err != nil || len(raw) != 16 {
		return ID{}, fmt.Errorf("domain: invalid id %q", s)
	}
	var id ID
	copy(id[:], raw)
	return id, nil
}

// IsValid reports whether the ID is a well-formed UUID v7.
func (id ID) IsValid() bool {
	return id.Version() == 7 && id[8]&0xc0 == 0x80
}

// Version returns the version nibble of the UUID (7 for UUID v7).
func (id ID) Version() int {
	return int(id[6] >> 4)
}

// IsZero reports whether the ID is the zero value.
func (id ID) IsZero() bool {
	return id == ID{}
}

// String returns the canonical 8-4-4-4-12 UUID representation.
func (id ID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], id[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], id[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], id[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], id[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], id[10:16])
	return string(buf[:])
}

// MarshalJSON encodes the ID as a quoted canonical UUID string.
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

// UnmarshalJSON decodes a quoted canonical UUID string into the ID.
func (id *ID) UnmarshalJSON(data []byte) error {
	if len(data) != 38 || data[0] != '"' || data[37] != '"' {
		return fmt.Errorf("domain: invalid id json %s", data)
	}
	parsed, err := ParseID(string(data[1:37]))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
