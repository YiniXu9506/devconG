package model

import (
	"database/sql"
	"fmt"
	"math/rand"
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
	ClickTime int64 `json:"click_time"`
}

// table `phrase_model` schema
type PhraseModel struct {
	PhraseID   int       `gorm:"primaryKey" json:"phrase_id"`
	Text       string    `json:"text"`
	GroupID    int       `json:"group_id"`
	OpenID     string    `json:"open_id"`
	Status     int       `json:"status"`
	CreateTime int64 `json:"create_time"`
	UpdateTime   int64 `json:"update_time"`
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

type PhraseUpdateTime struct {
	ClickTime int64 `json:"click_time"`
}

func getReturnPhraseCount(limit int, reviewedPhraseCount int, db *gorm.DB) (int, int, int) {
	if reviewedPhraseCount < limit {
		limit = reviewedPhraseCount
	}

	newestPhrasesCount := int(float64(limit) * 0.3)
	topNPhrasesCount := int(float64(limit) * 0.3)

	return newestPhrasesCount, topNPhrasesCount, limit
}

func (cp *CachePhrases) GetPhrasesList(limit int, db *gorm.DB) []PhraseItem {
	cp.mu.RLock()

	var phrase []PhraseItem

	reviewedPhraseCount := len(cp.PhraseList)
	newestPhrasesCount, topNPhrasesCount, limit := getReturnPhraseCount(limit, reviewedPhraseCount, db)
	randeomPhraseCount := limit - newestPhrasesCount - topNPhrasesCount

	// percentGap uses to slice phrase
	sliceGap := int(float64(reviewedPhraseCount) * 0.3)

	phrase = append(phrase, cp.PhraseList[:newestPhrasesCount]...)
	phrase = append(phrase, cp.PhraseList[sliceGap:sliceGap+topNPhrasesCount]...)
	phrase = append(phrase, cp.PhraseList[2*sliceGap:2*sliceGap+randeomPhraseCount]...)

	// fmt.Printf("%v\n", cp.PhraseList[:newestPhrasesCount])
	// fmt.Printf("%v\n", cp.PhraseList[sliceGap: sliceGap + topNPhrasesCount])
	// fmt.Printf("%v\n", cp.PhraseList[2 * sliceGap: 2 * sliceGap +randeomPhraseCount])
	rand.Shuffle(len(phrase), func(i, j int) {
		phrase[i], phrase[j] = phrase[j], phrase[i]
	})

	defer cp.mu.RUnlock()

	return phrase
}

/* calculate and update phrases
append 30% neweset phrases, whose status need to be reviewd
append 30% hot phrases
append 40% random phrases
*/

func (cp *CachePhrases) updateStats(db *gorm.DB) {
	cp.mu.Lock()

	type topPhraseItem struct {
		Clicks   int `json:"clicks"`
		PhraseID int `json:"phrase_id"`
		GroupID  int `json:"group_id"`
	}

	var allPhrasesItems []PhraseItem
	var newestPhrases, randomPhrases []PhraseModel
	var topPhrases []topPhraseItem

	var reviewedPhraseCount int
	limit := 100
	db.Raw("Select count(*) from phrase_models where status=1").Find(&reviewedPhraseCount)
	newestPhrasesCount, topNPhrasesCount, limit := getReturnPhraseCount(limit, reviewedPhraseCount, db)

	// get newest-N phrases
	newestPhrasesRes := db.Table("phrase_models").Where("status = ?", 2).Order("update_time desc").Limit(newestPhrasesCount).Find(&newestPhrases)

	// get top-N click phrases
	topNPhrasesRes := db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_click_models as a LEFT JOIN phrase_models as b ON a.phrase_id = b.phrase_id and b.status = 1 group by a.phrase_id order by clicks desc limit @limit", sql.Named("limit", topNPhrasesCount)).Scan(&topPhrases)

	// de-duplicate phrase
	allIDs := make(map[int]bool)
	var allIDSorted []int
	for _, item := range newestPhrases {
		if res, ok := allIDs[item.PhraseID]; !ok || !res {
			allIDSorted = append(allIDSorted, item.PhraseID)
		}

		allIDs[item.PhraseID] = true
	}
	for _, item := range topPhrases {
		if res, ok := allIDs[item.PhraseID]; !ok || !res {
			allIDSorted = append(allIDSorted, item.PhraseID)
		}
		allIDs[item.PhraseID] = true
	}

	// get more random phrase if de-duplicate topNPhrases and newestPhrases
	for len(allIDs) < limit {
		delta := limit - len(allIDs)
		randomPhrasesRes := db.Raw("SELECT * FROM phrase_models where status = 2 ORDER BY RAND() LIMIT ?", delta).Scan(&randomPhrases)
		if randomPhrasesRes.Error != nil {
			fmt.Printf("error")
			return
		}
		for _, item := range randomPhrases {
			if res, ok := allIDs[item.PhraseID]; !ok || !res {
				allIDSorted = append(allIDSorted, item.PhraseID)
			}
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

	for _, id := range allIDSorted {
		var phrase PhraseItem
		var phraseRecord PhraseModel
		var topClickPhrases topPhraseItem
		var topClickGroup topPhraseItem
		var phraseUpdateTime PhraseUpdateTime

		db.Table("phrase_models").Select("phrase_id, text, group_id").Where("phrase_id = ?", id).Find(&phrase)
		db.Table("phrase_models").Select("group_id").Where("phrase_id = ?", id).Find(&phraseRecord)

		// find out which group has top clicks on specific phrase
		db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id, group_id").Where("phrase_id = ?", id).Group("phrase_id, group_id").Order("clicks desc").Limit(1).Find(&topClickGroup)

		if topClickGroup.GroupID == 0 {
			topClickGroup.GroupID = phraseRecord.GroupID
		} else {
			// find out all click counts of a specific phrase
			db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id").Where("phrase_id = ?", id).Group("phrase_id").Find(&topClickPhrases)

			// update phrase click time
			db.Table("phrase_click_models").Select("click_time").Where("phrase_id = ?", id).Order("click_time desc").Limit(1).Find(&phraseUpdateTime)
			db.Table("phrase_models").Where("phrase_id = ?", id).Update("update_time", phraseUpdateTime.ClickTime)
		}

		phrase.Clicks = topClickPhrases.Clicks
		phrase.HotGroupClicks = topClickGroup.Clicks
		phrase.HotGroupID = topClickGroup.GroupID
		allPhrasesItems = append(allPhrasesItems, phrase)
	}

	cp.PhraseList = allPhrasesItems
	cp.mu.Unlock()
}

func UpdateStats(db *gorm.DB, cache *CachePhrases) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		<-ticker.C
		cache.updateStats(db)
	}
}
