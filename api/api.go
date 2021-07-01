package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type phrase struct {
	PhraseID       int    `json:"phrase_id"`
	Text           string `json:"text"`
	Clicks         int    `json:"clicks"`
	HotGroupID     int    `json:"hot_group_id"`
	HotGroupClicks int    `json:"hot_group_clicks"`
}

type latestClickedPhrase struct {
	PhraseID int    `json:"phrase_id"`
	Clicks   int    `json:"clicks"`
	OpenID   string `json:"open_id"`
	GroupID  int    `json:"group_id"`
}

// get all phrases
func GetPhrases(c *gin.Context, db *gorm.DB) {
	var phraseList []phrase
	const defaultLimit = "20"
	limit, err := strconv.Atoi(c.DefaultPostForm("limit", defaultLimit))
	if err != nil {
		fmt.Printf("failed to convert string to int")
		limit = 100
	}
	res := db.Table("phrase_models").Select("phrase_id", "text", "hot_group_id", "hot_group_clicks", "clicks").Limit(limit).Find(&phraseList)
	if res.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"m": "not found",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": phraseList,
		"m": "",
	})
}

// add a new phrase
func AddPhrase(c *gin.Context, db *gorm.DB) {
	var resDB model.PhraseModel
	text := c.PostForm("text")
	open_id := c.PostForm("open_id")
	group_id, _ := strconv.Atoi(c.PostForm("group_id"))
	isValidate := utils.ValidateText(text)

	fmt.Printf("text %v\n", text)
	res := db.Where(&model.PhraseModel{Text: text}).Find(&resDB)

	fmt.Printf("RowsAffected %v", res.RowsAffected)

	if isValidate {
		if res.RowsAffected == 0 {
			res1 := db.Create(&model.PhraseModel{Text: text, OpenID: open_id, GroupID: group_id})
			if res1.Error != nil {
				fmt.Printf("Error %v", res1.Error)
			}

			c.JSON(http.StatusOK, gin.H{
				"c": 0,
				"d": "",
				"m": "",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"c": 10001,
				"d": "",
				"m": "An existing item already exists",
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"c": 10002,
			"d": "",
			"m": "Maximum 10 characters",
		})
	}
}

func UpdateStats(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		var allPhrases []model.PhraseModel
		db.Table("phrase_models").Find(&allPhrases)
		type Result struct {
			GroupID  int `json:"group_id"`
			PhraseID int `json:"phrase_id"`
			Clicks   int `json:"clicks"`
		}

		for _, phrase := range allPhrases {
			var result Result
			phraseID := phrase.PhraseID
			res1 := db.Table("phrase_click_models").
				Select("phrase_id, group_id, sum(clicks) as clicks").
				Where("phrase_id = ?", phraseID).
				Group("phrase_id, group_id").
				Order("clicks desc").Limit(1).Find(&result)

			var sumClicks int
			res2 := db.Table("phrase_click_models").
				Select("sum(clicks) as clicks").
				Where("phrase_id = ?", phraseID).Find(&sumClicks)
			if res1.RowsAffected > 0 && res2.RowsAffected > 0 {
				phrase.Clicks = sumClicks
				phrase.HotGroupID = result.GroupID
				phrase.HotGroupClicks = result.Clicks
				phrase.UpdateTime = time.Now().Unix()
				db.Save(&phrase)
			}
		}
	}
}

// update phrase click counts
func UpdateClickedPhrase(c *gin.Context, db *gorm.DB) {
	var latestClickedPhrases []latestClickedPhrase

	// bind json
	if err := c.ShouldBindJSON(&latestClickedPhrases); err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Res %v", latestClickedPhrases)
	var phraseIDs []int
	for _, phrase := range latestClickedPhrases {
		var resDB model.PhraseModel
		phrase_id := phrase.PhraseID
		phraseIDs = append(phraseIDs, phrase_id)
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		fmt.Printf("phrase %v\n %v\n %v\n %v\n", phrase_id, clicks, open_id, group_id)

		res := db.Table("phrase_models").Where(&model.PhraseModel{PhraseID: phrase_id}).Find(&resDB)

		if res.RowsAffected > 0 {
			res1 := db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now()})
			if res1.Error != nil {
				fmt.Printf("Error %v", res1.Error)
			}
		}
	}

	// type Result struct {
	// 	GroupID  int `json:"group_id"`
	// 	PhraseID int `json:"phrase_id"`
	// 	Clicks   int `json:"clicks"`
	// }

	// var resultList []Result

	// db.Table("phrase_click_models").Select("phrase_id, group_id, sum(clicks) as clicks").Group("phrase_id, group_id").Order("clicks desc").Scan(&resultList)

	// fmt.Printf("result %v\n %v\n", resultList, len(resultList))

	type Return struct {
		PhraseID       int    `json:"phrase_id"`
		Text           string `json:"text"`
		HotGroupID     int    `json:"hot_group_id"`
		HotGroupClicks int    `json:"hot_group_clicks"`
		Clicks         int    `json:"clicks"`
		UpdateTime     int64  `json:"update_time"`
	}
	var res []Return
	for _, id := range phraseIDs {
		var item Return
		r := db.Table("phrase_models").Select("phrase_id, text, hot_group_id, hot_group_clicks, clicks, update_time").Where("phrase_id = ?", id).Scan(&item)
		if r.RowsAffected > 0 {
			res = append(res, item)
		}
	}
	// modelReturn := []Return{
	// 	{
	// 		PhraseID:      1,
	// 		Text:          "tidb1",
	// 		HotGroupID:    3,
	// 		HotGroupClick: 40,
	// 		Clicks:        100,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      2,
	// 		Text:          "tidb2",
	// 		HotGroupID:    2,
	// 		HotGroupClick: 45,
	// 		Clicks:        120,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      3,
	// 		Text:          "tidb3",
	// 		HotGroupID:    2,
	// 		HotGroupClick: 10,
	// 		Clicks:        80,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      4,
	// 		Text:          "tidb4",
	// 		HotGroupID:    8,
	// 		HotGroupClick: 49,
	// 		Clicks:        100,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      5,
	// 		Text:          "tidb5",
	// 		HotGroupID:    3,
	// 		HotGroupClick: 40,
	// 		Clicks:        100,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      6,
	// 		Text:          "tidb6",
	// 		HotGroupID:    2,
	// 		HotGroupClick: 20,
	// 		Clicks:        60,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      7,
	// 		Text:          "tidb7",
	// 		HotGroupID:    5,
	// 		HotGroupClick: 25,
	// 		Clicks:        80,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// 	{
	// 		PhraseID:      8,
	// 		Text:          "tidb8",
	// 		HotGroupID:    1,
	// 		HotGroupClick: 15,
	// 		Clicks:        30,
	// 		UpdateTime:    time.Now().Unix(),
	// 	},
	// }

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": res,
		"m": "",
	})
}
