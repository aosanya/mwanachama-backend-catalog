package models

import (
	"crypto/sha256"
	"encoding/hex"
)

// ShareLink is one key that opens a private Entry. The raw key is returned
// once, when the link is issued, and never stored — KeyHash is all the
// store keeps.
//
// An Entry may have many, and rotating issues a new one and retires the
// old rather than overwriting it, so who was handed what, and when it
// stopped working, survives the rotation.
type ShareLink struct {
	// ID is the storage key, opaque to callers.
	ID string `json:"id"`

	// Slug is the Entry this link opens.
	Slug string `json:"slug"`

	// KeyHash is the SHA-256 of the raw key, hex-encoded.
	KeyHash string `json:"-"`

	// Label is an optional note about who this link went to.
	Label string `json:"label,omitempty"`

	// Status is whether the link still opens the Entry.
	Status LinkStatus `json:"status"`

	// CreatedBy names whoever issued it.
	CreatedBy string `json:"created_by,omitempty"`

	// CreatedAt is the RFC 3339 timestamp when the link was issued.
	CreatedAt string `json:"created_at"`

	// RetiredAt is when it stopped working, empty while active.
	RetiredAt string `json:"retired_at,omitempty"`
}

// HashKey returns the stored form of a raw share key.
func HashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
