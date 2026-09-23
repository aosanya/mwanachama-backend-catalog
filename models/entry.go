package models

import (
	"encoding/json"
	"regexp"
	"strings"
)

// SlugPattern is the shape of an Entry's Slug: lowercase alphanumeric words
// joined by single hyphens. It is the URL key, so it is fixed at creation
// and never changes.
var SlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Entry is one catalog document — a mined organizational archetype, a
// library's title record, or whatever else a domain keeps. Doc is the whole
// thing.
//
// Only Slug, Name and Summary sit beside it, because only those three mean
// the same in every domain. Anything domain-shaped — an agency's sector and
// archetype, a library's genre and format — stays inside Doc and is filtered
// on by JSON path, so this type never learns one domain's vocabulary.
type Entry struct {
	// ID is the storage key, opaque to callers.
	ID string `json:"id"`

	// Slug is the stable URL key, unique across the catalog and immutable
	// after creation.
	Slug string `json:"slug"`

	// Name is the document's human-readable title.
	Name string `json:"name"`

	// Summary is the card's one-paragraph description.
	Summary string `json:"summary,omitempty"`

	// Doc is the whole document, as JSON text. Stored as jsonb wherever the
	// database has it, so a caller can filter and count inside it in the
	// database rather than parsing every row.
	Doc string `json:"doc"`

	// Visibility is who may read Doc.
	Visibility Visibility `json:"visibility"`

	// SubmittedBy names whatever wrote this revision — a mining run, or a
	// person.
	SubmittedBy string `json:"submitted_by,omitempty"`

	// ApprovedAt is when it was last moved off pending. Cleared whenever
	// Doc changes, because the reviewed thing no longer exists.
	ApprovedAt string `json:"approved_at,omitempty"`

	// CreatedAt is the RFC 3339 timestamp when this Entry was created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt is the RFC 3339 timestamp of the last update.
	UpdatedAt string `json:"updated_at"`

	// Deleted marks a soft-deleted Entry.
	Deleted bool `json:"deleted"`
}

// Validate reports whether e carries the fields the store requires: a
// well-formed Slug, a Name, a recognized Visibility, and a Doc that is a
// JSON object rather than an array, a scalar or nothing.
func (e Entry) Validate() bool {
	if !SlugPattern.MatchString(e.Slug) || strings.TrimSpace(e.Name) == "" {
		return false
	}
	if !IsVisibility(e.Visibility) {
		return false
	}
	return IsJSONObject(e.Doc)
}

// IsJSONObject reports whether s parses as a JSON object. The store keeps
// documents, never bare arrays or scalars, and a private Entry whose Doc
// failed to parse would be indistinguishable from one that was never
// written.
func IsJSONObject(s string) bool {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return false
	}
	_, ok := v.(map[string]any)
	return ok
}
