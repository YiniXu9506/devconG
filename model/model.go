package model

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
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
	PhraseID   int       `gorm:"primaryKey" json:"phrase_id"`
	Text       string    `json:"text"`
	GroupID    int       `json:"group_id"`
	OpenID     string    `json:"open_id"`
	Status     int       `json:"status"`
	CreateTime time.Time `json:"create_time"`
	ShowTime   time.Time `json:"show_time"`
}

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

func (cp *CachePhrases) GetAllItems() []PhraseItem {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return cp.PhraseList
}

func (cp *CachePhrases) updateStats(db *gorm.DB) {
	cp.mu.Lock()
	var allPhrasesItems []PhraseItem
	var newestPhrases, randomPhrases []PhraseModel
	type shortItem struct {
		Clicks   int `json:"clicks"`
		PhraseID int `json:"phrase_id"`
		GroupID  int `json:"group_id"`
	}
	var topNItems []shortItem
	var total int
	db.Raw("Select count(*) from phrase_models where status=1").First(&total)

	//  topNPhrases, randomPhrases []PhraseModel
	// var count int
	// db.Row
	// 0 -- not show
	// 1 -- show
	// 2 -- delete
	newestPhrasesRes := db.Table("phrase_models").Where("status = ?", 1).Order("show_time").Limit(10).Find(&newestPhrases)

	topNPhrasesRes := db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_click_models as a LEFT JOIN phrase_models as b ON a.phrase_id = b.phrase_id and b.status = 1 group by a.phrase_id order by clicks desc limit 10;").Scan(&topNItems)

	allIDs := make(map[int]bool)
	for _, item := range newestPhrases {
		allIDs[item.PhraseID] = true
	}
	for _, item := range topNItems {
		allIDs[item.PhraseID] = true
	}
	expectNum := 50
	if expectNum > total {
		expectNum = total
	}

	for len(allIDs) < expectNum {
		detal := expectNum - len(allIDs)
		randomPhrasesRes := db.Raw("SELECT * FROM phrase_models where status = 1 ORDER BY RAND() LIMIT ?", detal).Scan(&randomPhrases)
		if randomPhrasesRes.Error != nil {
			fmt.Printf("error")
			return
		}
		for _, item := range randomPhrases {
			allIDs[item.PhraseID] = true
		}
	}
	fmt.Println("xxx:", len(allIDs), total)
	if newestPhrasesRes.Error != nil {
		fmt.Printf("error")
		return
	}

	if topNPhrasesRes.Error != nil {
		fmt.Printf("error")
		return
	}

	for id := range allIDs {
		var temp PhraseItem
		var countTemp shortItem
		var countTemp2 shortItem
		db.Table("phrase_models").Select("phrase_id, text").Where("phrase_id = ?", id).Find(&temp)
		db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id").Where("phrase_id = ?", id).Group("phrase_id").Find(&countTemp)
		db.Table("phrase_click_models").Select("sum(clicks) as clicks, phrase_id, group_id").Where("phrase_id = ?", id).Group("phrase_id, group_id").Order("clicks desc").Limit(1).Find(&countTemp2)

		temp.Clicks = countTemp.Clicks
		temp.HotGroupClicks = countTemp2.Clicks
		temp.HotGroupID = countTemp2.GroupID
		allPhrasesItems = append(allPhrasesItems, temp)
		db.Table("phrase_models").Where("phrase_id = ?", id).Update("show_time", time.Now())
	}

	//fmt.Printf("allPhrases\n %v\n", allPhrases)

	// fmt.Printf("newestPhrasesRes\n %#+v\n", allPhrasesItems)
	cp.PhraseList = allPhrasesItems
	cp.mu.Unlock()
}

func UpdateStats(db *gorm.DB, cache *CachePhrases) {
	ticker := time.NewTicker(3 * time.Second)
	fmt.Printf("ticker %v\n", ticker)
	for {
		<-ticker.C
		cache.updateStats(db)
	}
}

// select * from pharse_models order by show_time desc limit 10;
// -> 最新 10;

// SELECT
// 	*
// FROM
// 	pharse_models
// ORDER BY RAND()
// LIMIT 50;

// -> random 50

//res2 := db.Table("phrase_click_models").
// Select("sum(clicks) as clicks", "phrase_id").
// Order(clicks).Limit(40).Find(&sumClicks)
// -> top 40
