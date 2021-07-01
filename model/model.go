package model

import (
	"time"
)

type PhraseClickModel struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	GroupID   int       `json:"group_id"`
	OpenID    string    `json:"open_id"`
	PhraseID  int       `json:"phrase_id"`
	Clicks    int       `json:"clicks"`
	ClickTime time.Time `json:"click_time"`
}

type PhraseModel struct {
	PhraseID       int       `gorm:"primaryKey" json:"phrase_id"`
	Text           string    `json:"text"`
	GroupID        int       `json:"group_id"`
	OpenID         string    `json:"open_id"`
	IsShow         bool      `json:"is_show"`
	IsDelete       bool      `json:"is_delete"`
	CreateTime     time.Time `json:"create_time"`
	Clicks         int       `json:"clicks"`
	HotGroupID     int       `json:"hot_group_id"`
	HotGroupClicks int       `json:"hot_group_clicks"`
	UpdateTime     int64     `json: "update_time"`
}
