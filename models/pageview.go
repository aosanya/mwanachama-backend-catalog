package models

// PageView is a running count for one page: the front page, or one Entry's
// page. A total only — there are no per-day rows, so nothing here can answer
// a question about a trend.
type PageView struct {
	// Page is which kind of page was viewed.
	Page PageKind `json:"page"`

	// Slug is the Entry viewed, empty for the front page.
	Slug string `json:"slug,omitempty"`

	// Views is the running total.
	Views int64 `json:"views"`
}

// Validate reports whether v is storable: a recognized page, and a slug
// present exactly when the page is an Entry's.
func (v PageView) Validate() bool {
	if !IsPageKind(v.Page) || v.Views < 0 {
		return false
	}
	if v.Page == PageKindIndex {
		return v.Slug == ""
	}
	return SlugPattern.MatchString(v.Slug)
}
