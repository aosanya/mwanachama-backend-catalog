package models

type RaisedItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Topic       string `json:"topic"`
	Publishable bool   `json:"publishable"`
	RaisedBy    string `json:"raised_by"`
	FiledBy     string `json:"filed_by"`
	Channel     string `json:"channel"`
	BoardItemID string `json:"board_item_id"`
	CreatedAt   string `json:"created_at"`
}

func (r RaisedItem) OnBehalfIsWhole() bool {
	return (r.FiledBy == "") == (r.Channel == "")
}
