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

// SlugFitsPage reports whether the slug is present exactly when the page is
// one Entry's. A spec can declare which pages exist and what a slug looks
// like; it cannot say that one field's presence depends on another's value,
// so this rule stays here.
func (v PageView) SlugFitsPage() bool {
	if v.Page == PageKindIndex {
		return v.Slug == ""
	}
	return SlugPattern.MatchString(v.Slug)
}
