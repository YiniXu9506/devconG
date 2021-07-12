package model

// table `phrase_click_model` schema
type PhraseClickModel struct {
	ID      int    `gorm:"primaryKey" json:"id"`
	GroupID int    `json:"group_id"`
	OpenID  string `json:"open_id"`
	// TODO: add index for phraseID
	PhraseID  int   `gorm:"index" json:"phrase_id"`
	Clicks    int   `json:"clicks"`
	ClickTime int64 `json:"click_time"`
}

// table `phrase_model` schema
type PhraseModel struct {
	PhraseID   int    `gorm:"primaryKey" json:"phrase_id"`
	Text       string `gorm:"uniqueIndex:text;size:60" json:"text"`
	GroupID    int    `json:"group_id"`
	OpenID     string `json:"open_id"`
	Status     int    `json:"status"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}
