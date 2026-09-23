package models

// MiningRun is one run of the routine that fills the catalog: its own
// report of what it did.
type MiningRun struct {
	// ID is the storage key.
	ID string `json:"id"`

	// RunID is the run's own identifier, as the routine knows it.
	RunID string `json:"run_id,omitempty"`

	// SubmittedBy names whatever reported it.
	SubmittedBy string `json:"submitted_by,omitempty"`

	// TemplatesFiled is how many Entries the run wrote.
	TemplatesFiled int `json:"templates_filed"`

	// Summary is the run's report.
	Summary string `json:"summary"`

	// CreatedAt is the RFC 3339 timestamp when it was recorded.
	CreatedAt string `json:"created_at"`
}

// Setting is one run-time knob the routine reads, such as the rule it ranks
// the catalog by. A key/value store rather than columns because the routine
// and the operator add knobs faster than a schema should change.
type Setting struct {
	// Key names the setting.
	Key string `json:"key"`

	// Value is its current value.
	Value string `json:"value"`

	// UpdatedAt is the RFC 3339 timestamp of the last write.
	UpdatedAt string `json:"updated_at"`
}
