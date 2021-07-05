package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// type phrase struct {
// 	PhraseID       int    `json:"phrase_id"`
// 	Text           string `json:"text"`
// 	Clicks         int    `json:"clicks"`
// 	HotGroupID     int    `json:"hot_group_id"`
// 	HotGroupClicks int    `json:"hot_group_clicks"`
// }

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

// get all phrases
func GetPhrases(c *gin.Context, db *gorm.DB, cachePhrases *model.CachePhrases) {
	fmt.Println("get api")
	// var phraseList []phrase
	// const defaultLimit = "100"
	// limit, err := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	// if err != nil {
	// 	fmt.Printf("failed to convert string to int")
	// 	limit = 100
	// }

	// TODD: calculate hot phrase group_id and clicks from phrase_clicks_models

	// res := db.Debug().Table("phrase_models").Select("phrase_id", "text", "hot_group_id", "hot_group_clicks", "clicks").Limit(limit).Find(&phraseList)
	// if res.Error != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"c": "1",
	// 		"d": "",
	// 		"m": "Fetch all phrases failed",
	// 	})
	// 	return
	// }

	items := cachePhrases.GetAllItems()
	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": items,
		"m": "",
	})
}

// add a new phrase
func AddPhrase(c *gin.Context, db *gorm.DB) {
	fmt.Println("add api")
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
	findRes := db.Debug().Where(&model.PhraseModel{Text: newPhrase.Text}).Find(&phrase)

	if isValidate {
		if findRes.RowsAffected == 0 {
			ceateRes := db.Debug().Table("phrase_models").Create(&model.PhraseModel{Text: newPhrase.Text, OpenID: newPhrase.OpenID, GroupID: newPhrase.GroupID, CreateTime: time.Now(), ShowTime: time.Now()})
			if ceateRes.Error != nil {
				fmt.Printf("Insert new phrase failed, %v", ceateRes.Error)
			}
			fmt.Printf("Insert new phrase success, %v, %v\n", phrase, newPhrase)

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
	fmt.Println("upadte api")
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
		var resDB model.PhraseModel
		phrase_id := phrase.PhraseID
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		fmt.Printf("phrase %v\n %v\n %v\n %v\n", phrase_id, clicks, open_id, group_id)

		res := db.Debug().Table("phrase_models").Where(&model.PhraseModel{PhraseID: phrase_id}).Find(&resDB)

		if res.RowsAffected > 0 {
			res1 := db.Debug().Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now()})
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
