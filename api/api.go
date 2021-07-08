package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"database/sql"

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

type distribution struct {
	GroupID int `json:"group_id"`
	Clicks  int `json:"clicks"`
}

type allPhrasesWithDistribution struct {
	model.PhraseModel
	Distributions []distribution `json:"distributions"`
}

type topNPhrasesWithDistribution struct {
	PhraseID      int            `json:"phrase_id"`
	Text          string         `json:"text"`
	Distributions []distribution `json:"distributions"`
}

type PagiInfo struct {
	Total int `json:"total"`
	Offset int `json:"offset"`
}

type allPhrasesWithDistributionResponse struct {
	Pagi PagiInfo `json:"pagi"`
	List []allPhrasesWithDistribution `json:"list"`
}

// return phrases to wechat
func GetSrollingPhrases(c *gin.Context, db *gorm.DB, cachePhrases *model.CachePhrases) {
	const defaultLimit = "100"
	limit, err := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	if err != nil {
		fmt.Printf("failed to convert string to int")
		limit = 100
	}

	items := cachePhrases.GetPhrasesList(limit, db)

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
			ceateRes := db.Table("phrase_models").Create(&model.PhraseModel{Text: newPhrase.Text, OpenID: newPhrase.OpenID, GroupID: newPhrase.GroupID, Status: 1, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()})
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
			res1 := db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now().Unix()})
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
func GetAllPhrases(c *gin.Context, db *gorm.DB) {
	defaultLimit := "50"
	defaultOffset := "0"
	defaultStatus := "1,2"

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", defaultOffset))
	status := c.DefaultQuery("status", defaultStatus)
	var finalStatus [3]interface{}

	s := strings.Split(status, ",")

	for i := 0; i < 3; i++ {
		if i < len(s) {
			finalStatus[i] = s[i]
		} else {
			finalStatus[i] = 0
		}
	}

	var phraseList []model.PhraseModel

	var distributions []distribution

	var phrasesWithDistribution []allPhrasesWithDistribution

	var allPhrasesResponse allPhrasesWithDistributionResponse

	var returnTotalCount []model.PhraseModel

	db.Table("phrase_models").Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", finalStatus[0]), sql.Named("status2", finalStatus[1]), sql.Named("status3", finalStatus[2])).Order("create_time desc").Limit(limit).Find(&returnTotalCount)
	db.Table("phrase_models").Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", finalStatus[0]), sql.Named("status2", finalStatus[1]), sql.Named("status3", finalStatus[2])).Order("create_time desc").Limit(limit).Offset(offset).Find(&phraseList)

	allPhrasesResponse.Pagi.Total = len(returnTotalCount)
	allPhrasesResponse.Pagi.Offset = offset

	for _, phrase := range phraseList {
		var temp allPhrasesWithDistribution
		db.Table("phrase_click_models").Select("group_id, SUM(clicks) as clicks").Where("phrase_id = ?", phrase.PhraseID).Group("group_id").Find(&distributions)

		temp.PhraseID = phrase.PhraseID
		temp.Text = phrase.Text
		temp.GroupID = phrase.GroupID
		temp.OpenID = phrase.OpenID
		temp.Status = phrase.Status
		temp.CreateTime = phrase.CreateTime
		temp.UpdateTime = phrase.UpdateTime

		distributionIDs := make(map[int]bool)

		for _, dist := range distributions {
			distributionIDs[dist.GroupID] = true
		}

		for i := 0; i < 5; i++ {
			if _, ok := distributionIDs[i+1]; !ok {
				var temp distribution
				temp.GroupID = i+1
				temp.Clicks = 0
				distributions = append(distributions,temp)
			}
		}

		temp.Distributions = distributions

		phrasesWithDistribution = append(phrasesWithDistribution, temp)
	}

	allPhrasesResponse.List = phrasesWithDistribution

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": allPhrasesResponse,
		"m": "",
	})
}

// get top-N phrases
func GetTopNPhrases(c *gin.Context, db *gorm.DB) {
	defaultLimit := "5"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	type topPhraseID struct {
		PhraseID int `json:"phrase_id"`
		Clicks   int `json:"clicks"`
	}

	type topPhraseText struct {
		Text       string    `json:"text"`
	}

	var topPhraseIDList []topPhraseID
	var topNPhrasesWithDistributions []topNPhrasesWithDistribution

	db.Table("phrase_click_models").Select("phrase_id, SUM(clicks) as clicks").Group("phrase_id").Order("clicks desc").Limit(limit).Find(&topPhraseIDList)

	for _, phrase := range topPhraseIDList {
		var distributions []distribution
		var topPhraseTextItem topPhraseText
		var temp topNPhrasesWithDistribution

		db.Table("phrase_click_models").Select("group_id, SUM(clicks) as clicks").Where("phrase_id = ?", phrase.PhraseID).Group("group_id").Find(&distributions)
		db.Table("phrase_models").Select("text").Where("phrase_id = ?", phrase.PhraseID).Find(&topPhraseTextItem)

		temp.PhraseID = phrase.PhraseID
		temp.Text = topPhraseTextItem.Text

		distributionIDs := make(map[int]bool)

		for _, dist := range distributions {
			distributionIDs[dist.GroupID] = true
		}

		for i := 0; i < 5; i++ {
			if _, ok := distributionIDs[i+1]; !ok {
				var temp distribution
				temp.GroupID = i+1
				temp.Clicks = 0
				distributions = append(distributions,temp)
			}
		}

		temp.Distributions = distributions

		topNPhrasesWithDistributions = append(topNPhrasesWithDistributions, temp)
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": topNPhrasesWithDistributions,
		"m": "",
	})
}

// func DeletePhrase(c *gin.Context, db *gorm.DB) {
// 	type phraseIDQuery struct {
// 		PhraseID int `json:"phrase_id" binding:"required"`
// 	}

// 	var IDQuery phraseIDQuery

// 	if err := c.ShouldBindWith(&IDQuery, binding.Query); err == nil {
// 		c.JSON(http.StatusOK, gin.H{"message": "Booking dates are valid!"})
// 	} else {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 	}

// }
