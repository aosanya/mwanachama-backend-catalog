package models

type BoardItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Outcome   string `json:"outcome"`
	CreatedAt string `json:"created_at"`
}

type BoardItemTally struct {
	BoardItem
	RaisedBy int `json:"raised_by"`
	MeToo    int `json:"me_too"`
}
