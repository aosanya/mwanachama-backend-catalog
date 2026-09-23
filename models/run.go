package models

type MiningRun struct {
	ID             string `json:"id"`
	RunID          string `json:"run_id,omitempty"`
	SubmittedBy    string `json:"submitted_by,omitempty"`
	TemplatesFiled int    `json:"templates_filed"`
	Summary        string `json:"summary"`
	CreatedAt      string `json:"created_at"`
}

type Setting struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}
