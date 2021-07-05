package model

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// table `phrase_click_model` schema
type PhraseClickModel struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	GroupID   int       `json:"group_id"`
	OpenID    string    `json:"open_id"`
	PhraseID  int       `json:"phrase_id"`
	Clicks    int       `json:"clicks"`
	ClickTime time.Time `json:"click_time"`
}

// table `phrase_model` schema
type PhraseModel struct {
	PhraseID   int       `gorm:"primaryKey" json:"phrase_id"`
	Text       string    `json:"text"`
	GroupID    int       `json:"group_id"`
	OpenID     string    `json:"open_id"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"create_time"`
	ShowTime   time.Time `json:"show_time"`
}

// API `/phrases` return phraseItem
type PhraseItem struct {
	PhraseID       int    `json:"phrase_id"`
	Text           string `json:"text"`
	Clicks         int    `json:"clicks"`
	HotGroupID     int    `json:"hot_group_id"`
	HotGroupClicks int    `json:"hot_group_clicks"`
}

type CachePhrases struct {
	PhraseList []PhraseItem
	mu         sync.RWMutex
}

var limitPhrase = 100

func (cp *CachePhrases) GetPhrasesList(limit int) []PhraseItem {
	limitPhrase = limit
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.PhraseList
}

/* calculate and update phrases
append 30% neweset phrases, whose status need to be reviewd
append 30% hot phrases
append 40% random phrases
*/

func (cp *CachePhrases) updateStats(db *gorm.DB) {
	cp.mu.Lock()
	var allPhrasesItems []PhraseItem
	var newestPhrases, randomPhrases []PhraseModel
	newestPhrasesCount := int(float64(limitPhrase) * 0.3)
	topNPhrasesCount := int(float64(limitPhrase) * 0.3)

	type topItem struct {
		Clicks   int `json:"clicks"`
		PhraseID int `json:"phrase_id"`
		GroupID  int `json:"group_id"`
	}
	var topNItems []topItem

	// get count of reviewed phrase
	var total int
	db.Raw("Select count(*) from phrase_models where status=1").Find(&total)

	if total < limitPhrase {
		limitPhrase = total
	}

	// get 10 newest phrases
	newestPhrasesRes := db.Table("phrase_models").Where("status = ?", 1).Order("show_time desc").Limit(newestPhrasesCount).Find(&newestPhrases)

	// get 10 top click phrases
	topNPhrasesRes := db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_click_models as a LEFT JOIN phrase_models as b ON a.phrase_id = b.phrase_id and b.status = 1 group by a.phrase_id order by clicks desc limit @limit", sql.Named("limit", topNPhrasesCount)).Scan(&topNItems)

	// de-duplicate phrase
	allIDs := make(map[int]bool)
	for _, item := range newestPhrases {
		allIDs[item.PhraseID] = true
	}
	for _, item := range topNItems {
		allIDs[item.PhraseID] = true
	}

	// get more random phrase if de-duplicate topNPhrases and newestPhrases
	for len(allIDs) < limitPhrase {
		delta := limitPhrase - len(allIDs)
		randomPhrasesRes := db.Raw("SELECT * FROM phrase_models where status = 1 ORDER BY RAND() LIMIT ?", delta).Scan(&randomPhrases)
		if randomPhrasesRes.Error != nil {
			fmt.Printf("error")
			return
		}
		for _, item := range randomPhrases {
			allIDs[item.PhraseID] = true
		}
	}

	if newestPhrasesRes.Error != nil {
		fmt.Printf("error: %v", newestPhrasesRes.Error)
		return
	}

	if topNPhrasesRes.Error != nil {
		fmt.Printf("error: %v", topNPhrasesRes.Error)
		return
	}

	for id := range allIDs {
		var temp PhraseItem
		var countTemp topItem
		var countTemp2 topItem
		db.Table("phrase_models").Select("phrase_id, text").Where("phrase_id = ?", id).Find(&temp)

		// find out all click counts of a specific phrase
		db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id").Where("phrase_id = ?", id).Group("phrase_id").Find(&countTemp)

		// find out which group has top clicks on specific phrase
		db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id, group_id").Where("phrase_id = ?", id).Group("phrase_id, group_id").Order("clicks desc").Limit(1).Find(&countTemp2)

		temp.Clicks = countTemp.Clicks
		temp.HotGroupClicks = countTemp2.Clicks
		temp.HotGroupID = countTemp2.GroupID
		allPhrasesItems = append(allPhrasesItems, temp)

		// update show time of selected phrases
		db.Table("phrase_models").Where("phrase_id = ?", id).Update("show_time", time.Now())
	}

	cp.PhraseList = allPhrasesItems
	cp.mu.Unlock()
}

func UpdateStats(db *gorm.DB, cache *CachePhrases, limit int) {
	limitPhrase = limit
	ticker := time.NewTicker(3 * time.Second)
	for {
		<-ticker.C
		cache.updateStats(db)
	}
}
