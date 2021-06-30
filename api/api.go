package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/YiniXu9506/devconG/mock"
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
	var resDB mock.PhraseModel
	text := c.PostForm("text")
	open_id := c.PostForm("open_id")
	group_id, _ := strconv.Atoi(c.PostForm("group_id"))
	isValidate := utils.ValidateText(text)

	fmt.Printf("text %v\n", text)
	res := db.Where(&mock.PhraseModel{Text: text}).Find(&resDB)

	fmt.Printf("RowsAffected %v", res.RowsAffected)

	if isValidate {
		if res.RowsAffected == 0 {
			res1 := db.Create(&mock.PhraseModel{Text: text, OpenID: open_id, GroupID: group_id})
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
	for _, phrase := range latestClickedPhrases {
		var resDB mock.PhraseModel
		phrase_id := phrase.PhraseID
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		fmt.Printf("phrase %v\n %v\n %v\n %v\n", phrase_id, clicks, open_id, group_id)

		res := db.Table("phrase_models").Where(&mock.PhraseModel{PhraseID: phrase_id}).Find(&resDB)

		fmt.Printf("res.RowsAffected %v\n", res.RowsAffected)

		if res.RowsAffected > 0 {
			res1 := db.Create(&mock.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now()})
			if res1.Error != nil {
				fmt.Printf("Error %v", res1.Error)
			}
		}
	}

	type Result struct {
		GroupID  int `json:"group_id"`
		PhraseID int `json:"phrase_id"`
		Clicks   int `json:"clicks"`
	}

	var resultList []Result

	db.Table("phrase_click_models").Select("phrase_id, group_id, sum(clicks) as clicks").Group("phrase_id, group_id").Order("clicks desc").Scan(&resultList)

	fmt.Printf("result %v\n %v\n", resultList, len(resultList))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}
