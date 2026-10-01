package models

type EntryPrivate struct {
	Slug          string `json:"slug"`
	Readme        string `json:"readme,omitempty"`
	Sales         string `json:"sales,omitempty"`
	ModuleConfigs string `json:"module_configs,omitempty"`
	Appeal        string `json:"appeal,omitempty"`
	Rank          int    `json:"rank,omitempty"`
	Doc           string `json:"doc,omitempty"`
	UpdatedAt     string `json:"updated_at"`
}
