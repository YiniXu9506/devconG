package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type latestClickedPhrase struct {
	PhraseID int    `json:"phrase_id"`
	Clicks   int    `json:"clicks"`
	OpenID   string `json:"open_id"`
	GroupID  int    `json:"group_id"`
}

type newPhrase struct {
	Text    string `json:"text"`
	OpenID  string `json:"open_id"`
	GroupID int    `json:"group_id"`
}

// return phrases to wechat
func GetSrollingPhrases(c *gin.Context, db *gorm.DB, cachePhrases *model.CachePhrases) {
	const defaultLimit = "100"
	limit, err := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	if err != nil {
		fmt.Printf("failed to convert string to int")
		limit = 100
	}

	items := cachePhrases.GetPhrasesList(limit)
	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": items,
		"m": "",
	})
}

// add a new phrase
func AddPhrase(c *gin.Context, db *gorm.DB) {
	var newPhrase newPhrase
	// bind json
	if err := c.ShouldBindJSON(&newPhrase); err != nil {
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{"error": err.Error()})
		return
	}

	// check text maxium length
	isValidate := utils.ValidateText(newPhrase.Text)

	// check text uniqueness
	var phrase model.PhraseModel
	findRes := db.Where(&model.PhraseModel{Text: newPhrase.Text}).Find(&phrase)

	if isValidate {
		if findRes.RowsAffected == 0 {
			ceateRes := db.Table("phrase_models").Create(&model.PhraseModel{Text: newPhrase.Text, OpenID: newPhrase.OpenID, GroupID: newPhrase.GroupID, Status: 1, CreateTime: time.Now(), ShowTime: time.Now()})
			if ceateRes.Error != nil {
				fmt.Printf("Insert new phrase failed, %v", ceateRes.Error)
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

	for _, phrase := range latestClickedPhrases {
		var resDB model.PhraseModel
		phrase_id := phrase.PhraseID
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		res := db.Table("phrase_models").Where(&model.PhraseModel{PhraseID: phrase_id}).Find(&resDB)

		if res.RowsAffected > 0 {
			res1 := db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now()})
			if res1.Error != nil {
				fmt.Printf("Error %v", res1.Error)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// get all phrases
func GetAllPhrases(c *gin.Context) {
	defaultLimit := "50"
	defaultOffset := "0"
	defaultStatus := "1,2"

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))
	offset, _ := strconv.Atoi(c.DefaultQuery("limit", defaultOffset))
	status, _ := strconv.Atoi(c.DefaultQuery("limit", defaultStatus))



}
