package models

type BoardItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Outcome   string `json:"outcome"`
	Hidden    bool   `json:"hidden"`
	CreatedAt string `json:"created_at"`
}

type BoardItemTally struct {
	BoardItem
	RaisedBy int `json:"raised_by"`
	MeToo    int `json:"me_too"`
}

type SharedBoardItem struct {
	Title  string   `json:"title"`
	Theme  string   `json:"theme"`
	Area   string   `json:"area"`
	MeToo  int      `json:"me_too"`
	Status string   `json:"status"`
	Files  []string `json:"files"`
}
