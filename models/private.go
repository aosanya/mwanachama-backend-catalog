package models

// EntryPrivate is everything about an Entry that its document does not
// carry and no storefront reader may see: the operator's own notes, the
// per-module configuration, and where it sits in the running order.
//
// It is a separate aggregate rather than more columns on Entry because the
// two have different readers. Entry.Doc goes out to anyone the Visibility
// allows; nothing here ever does.
type EntryPrivate struct {
	// Slug is the Entry this belongs to.
	Slug string `json:"slug"`

	// Readme is the operator-facing description of the archetype.
	Readme string `json:"readme,omitempty"`

	// Sales is the prospect list and open bids, as Markdown. Empty
	// wherever an Entry is not something being sold.
	Sales string `json:"sales,omitempty"`

	// ModuleConfigs maps a module code to that module's configuration for
	// this archetype, as JSON text. Stored as jsonb wherever the database
	// has it.
	ModuleConfigs string `json:"module_configs,omitempty"`

	// Appeal is the one-line reason this archetype ranks where it does.
	Appeal string `json:"appeal,omitempty"`

	// Rank orders the catalog, lowest first. Zero means unranked, which
	// sorts last.
	Rank int `json:"rank,omitempty"`

	// UpdatedAt is the RFC 3339 timestamp of the last update.
	UpdatedAt string `json:"updated_at"`
}

// Validate reports whether p is storable: a well-formed Slug, a
// non-negative Rank, and ModuleConfigs that is either empty or a JSON
// object keyed by module code.
func (p EntryPrivate) Validate() bool {
	if !SlugPattern.MatchString(p.Slug) || p.Rank < 0 {
		return false
	}
	return p.ModuleConfigs == "" || IsJSONObject(p.ModuleConfigs)
}
