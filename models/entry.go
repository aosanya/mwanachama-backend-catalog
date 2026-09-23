package models

type Entry struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Summary     string     `json:"summary,omitempty"`
	Doc         string     `json:"doc"`
	Visibility  Visibility `json:"visibility"`
	SubmittedBy string     `json:"submitted_by,omitempty"`
	ApprovedAt  string     `json:"approved_at,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	Deleted     bool       `json:"deleted"`
}
