package model

// table `phrase_click_model` schema
type PhraseClickModel struct {
	ID        int    `gorm:"primaryKey" json:"id"`
	PhraseID  int    `gorm:"index:idx_phrase_click" json:"phrase_id"`
	GroupID   int    `gorm:"index:idx_phrase_click" json:"group_id"`
	OpenID    string `json:"open_id"`
	Clicks    int    `gorm:"index:idx_phrase_click" json:"clicks"`
	ClickTime int64  `json:"click_time"`
}

// table `phrase_model` schema
type PhraseModel struct {
	PhraseID   int    `gorm:"primaryKey" json:"phrase_id"`
	Text       string `gorm:"uniqueIndex:text;size:60" json:"text"`
	GroupID    int    `json:"group_id"`
	OpenID     string `json:"open_id"`
	Status     int    `gorm:"index" json:"status"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time"`
}

type UserModel struct {
	OpenID     string `gorm:"primaryKey" json:"open_id" binding:"required"`
	NickName   string `json:"nick_name"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	HeadImgURL string `json:"headimgurl"`
}
