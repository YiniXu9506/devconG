package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type distributionModel struct {
	GroupID int `json:"group_id"`
	Clicks  int `json:"clicks"`
}

type phraseWithDistributionModel struct {
	model.PhraseModel
	Distributions []distributionModel `json:"distributions"`
}

type topNPhrasesWithDistribution struct {
	PhraseID      int                 `json:"phrase_id"`
	Text          string              `json:"text"`
	Distributions []distributionModel `json:"distributions"`
}

type PagiInfo struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
}

type allPhraseResponse struct {
	Pagi PagiInfo                      `json:"pagi"`
	List []phraseWithDistributionModel `json:"list"`
}

func phraseDistribution(distributions []distributionModel) []distributionModel {
	distributionGroupIDs := make(map[int]bool)

	for _, dist := range distributions {
		distributionGroupIDs[dist.GroupID] = true
	}

	for i := 0; i < 5; i++ {
		if _, ok := distributionGroupIDs[i+1]; !ok {
			var phraseDistribution distributionModel
			phraseDistribution.GroupID = i + 1
			phraseDistribution.Clicks = 0
			distributions = append(distributions, phraseDistribution)
		}
	}

	return distributions
}

// return phrases to wechat
func (s *Service) GetScrollingPhrasesHandler(c *gin.Context) {
	const defaultLimit = "100"
	limit, err := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	if err != nil {
		fmt.Printf("failed to convert string to int")
		limit = 100
	}
	//TODO: error handling
	scrollingPhrasesRes := s.provider.GetScrollingPhrases(limit)

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": scrollingPhrasesRes,
		"m": "",
	})
}

// add a new phrase
func (s *Service) AddPhraseHandler(c *gin.Context) {
	type phraseRequest struct {
		Text    string `form:"text" json:"text" binding:"required"`
		OpenID  string `form:"open_id" json:"open_id" binding:"required"`
		GroupID int    `form:"group_id" json:"group_id" binding:"required"`
	}

	var req phraseRequest
	// bind json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "phrase_id, open_id, group_id are required!",
		})
		return
	}

	// check text maxium length
	isValidate := utils.ValidateText(req.Text)

	if !isValidate {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 10002,
			"d": "",
			"m": "Maximum 10 characters",
		})

		return
	}

	ceateRes := s.db.Table("phrase_models").Create(&model.PhraseModel{Text: req.Text, OpenID: req.OpenID, GroupID: req.GroupID, Status: 1, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()})
	if ceateRes.Error != nil {
		mysqlErr := &mysql.MySQLError{}
		if errors.As(ceateRes.Error, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusForbidden, gin.H{
				"c": 10001,
				"d": "",
				"m": ceateRes.Error.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": ceateRes.Error.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// update phrase click counts
func (s *Service) UpdateClickedPhraseHandler(c *gin.Context) {
	type latestClickedPhraseRequest struct {
		PhraseID int    `form:"phrase_id" json:"phrase_id" binding:"required"`
		Clicks   int    `form:"clicks" json:"clicks" binding:"required"`
		OpenID   string `form:"open_id" json:"open_id" binding:"required"`
		GroupID  int    `form:"group_id" json:"group_id" binding:"required"`
	}

	var req []latestClickedPhraseRequest

	// bind json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "phrase_id, clicks, open_id and group_id are required!",
		})
		return
	}

	for _, phrase := range req {
		var phraseRecord model.PhraseModel
		phrase_id := phrase.PhraseID
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		// check validation of phrase in phrase_models
		phraseRecordRe := s.db.Debug().Table("phrase_models").Where("phrase_id = ? AND status = ?", phrase_id, 2).Find(&phraseRecord)

		//fmt.Printf("phraseRecordCount %v\n", phraseRecord)
		//zap.L().Sugar().Debugf("phraseRecordCount %v\n", phraseRecord)

		if phraseRecordRe.RowsAffected > 0 {
			res := s.db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now().Unix()})
			if res.Error != nil {
				zap.L().Sugar().Infof("Failed to update phrase click model %v", res.Error)
				c.JSON(http.StatusInternalServerError, gin.H{
					"c": 1,
					"d": "",
					"m": res.Error.Error(),
				})
				return
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
func (s *Service) GetAllPhrasesHandler(c *gin.Context) {
	defaultLimit := "50"
	defaultOffset := "0"
	defaultStatus := "1,2"

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", defaultOffset))
	status := c.DefaultQuery("status", defaultStatus)

	var statusMap [3]interface{}
	str := strings.Split(status, ",")

	for i := 0; i < 3; i++ {
		if i < len(str) {
			statusMap[i] = str[i]
		} else {
			statusMap[i] = 0
		}
	}

	var phraseList []model.PhraseModel
	var distributions []distributionModel
	var allPhrasesWithDistributions []phraseWithDistributionModel

	var allPhrasesResp allPhraseResponse

	var phraseTotalCount int

	//TODO: error handling
	if err := s.db.Table("phrase_models").Select("count(*)").Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", statusMap[0]), sql.Named("status2", statusMap[1]), sql.Named("status3", statusMap[2])).Find(&phraseTotalCount).Error; err != nil {
		zap.L().Sugar().Errorf("select meet error %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": err.Error(),
		})
		return
	}
	//TODO: error handling
	s.db.Table("phrase_models").Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", statusMap[0]), sql.Named("status2", statusMap[1]), sql.Named("status3", statusMap[2])).Order("create_time desc").Limit(limit).Offset(offset).Find(&phraseList)

	allPhrasesResp.Pagi.Total = phraseTotalCount
	allPhrasesResp.Pagi.Offset = offset

	for _, phrase := range phraseList {
		var phraseWithDistribution phraseWithDistributionModel
		//TODO: error handling
		s.db.Table("phrase_click_models").Select("group_id, SUM(clicks) as clicks").Where("phrase_id = @phrase_id", sql.Named("phrase_id", phrase.PhraseID)).Group("group_id").Find(&distributions)
		phraseWithDistribution.PhraseID = phrase.PhraseID
		phraseWithDistribution.Text = phrase.Text
		phraseWithDistribution.GroupID = phrase.GroupID
		phraseWithDistribution.OpenID = phrase.OpenID
		phraseWithDistribution.Status = phrase.Status
		phraseWithDistribution.CreateTime = phrase.CreateTime
		phraseWithDistribution.UpdateTime = phrase.UpdateTime
		phraseWithDistribution.Distributions = phraseDistribution(distributions)

		allPhrasesWithDistributions = append(allPhrasesWithDistributions, phraseWithDistribution)
	}

	allPhrasesResp.List = allPhrasesWithDistributions

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": allPhrasesResp,
		"m": "",
	})
}

// get top-N phrases
func (s *Service) GetTopNPhrasesHandler(c *gin.Context) {
	defaultLimit := "5"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	type topPhraseID struct {
		PhraseID int `json:"phrase_id"`
		Clicks   int `json:"clicks"`
	}

	type textModel struct {
		Text string `json:"text"`
	}

	var topPhraseIDs []topPhraseID
	var topNPhrasesWithDistributions []topNPhrasesWithDistribution

	// get top N phrases, which are reviewed
	// WRONG!!
	//TODO: error handling
	s.db.Table("phrase_click_models").Select("phrase_id, SUM(clicks) as clicks").Group("phrase_id").Order("clicks desc").Limit(limit).Find(&topPhraseIDs)

	for _, phrase := range topPhraseIDs {
		var distributions []distributionModel
		var topPhraseText textModel
		var phraseWithDistribution topNPhrasesWithDistribution
		//TODO: error handling
		s.db.Table("phrase_click_models").Select("group_id, SUM(clicks) as clicks").Where("phrase_id = ?", phrase.PhraseID).Group("group_id").Find(&distributions)
		//TODO: error handling
		s.db.Table("phrase_models").Select("text").Where("phrase_id = ?", phrase.PhraseID).Find(&topPhraseText)

		phraseWithDistribution.PhraseID = phrase.PhraseID
		phraseWithDistribution.Text = topPhraseText.Text
		phraseWithDistribution.Distributions = phraseDistribution(distributions)

		topNPhrasesWithDistributions = append(topNPhrasesWithDistributions, phraseWithDistribution)
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": topNPhrasesWithDistributions,
		"m": "",
	})
}

// delete phrase by change status to 3
func (s *Service) DeletePhraseHandler(c *gin.Context) {
	type phraseIDRequest struct {
		PhraseID int `form:"id" json:"id" binding:"required"`
	}

	var req phraseIDRequest
	// var deletePhrase model.PhraseModel

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "phrase_id is required!",
		})
		return
	}
	//TODO: error handling
	deletePhraseRes := s.db.Table("phrase_models").Where("phrase_id = ?", req.PhraseID).Updates(map[string]interface{}{"status": 3, "update_time": time.Now().Unix()})
	if deletePhraseRes.Error != nil {
		zap.L().Sugar().Error("delete meet error", deletePhraseRes.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": deletePhraseRes.Error.Error(),
		})
		return
	}
	if deletePhraseRes.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{
			"c": 11001,
			"d": "",
			"m": "Nonexistent",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// update phrase text or status
func (s *Service) PatchPhraseHandler(c *gin.Context) {
	type patchPhraseReq struct {
		PhraseID int    `form:"id" json:"id" binding:"required"`
		Text     string `form:"text" json:"text"`
		Status   int    `form:"status" json:"status"`
	}

	var req patchPhraseReq
	var row model.PhraseModel

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "phrase_id is required!",
		})
		return
	}
	// refine error structure

	// check whether the phrase exist or not
	// TODO: error handling
	phraseRes := s.db.Table("phrase_models").Where("phrase_id = ?", req.PhraseID).Find(&row)
	// phraseRes.Error != nil{}

	if phraseRes.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{
			"c": 0,
			"d": "",
			"m": "This phrase does not exist",
		})
		return
	}

	isValidate := utils.ValidateText(req.Text)

	// update text of phrase
	updates := make(map[string]interface{})
	if isValidate {
		updates["text"] = req.Text
		updates["update_time"] = time.Now().Unix()
	}
	// update status of phrase
	if req.Status > 0 && req.Status <= 3 {
		updates["status"] = req.Status
		updates["update_time"] = time.Now().Unix()
	}
	if len(updates) > 0 {
		s.db.Table("phrase_models").Where("phrase_id = ?", req.PhraseID).Updates(updates)
	}
	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// get phrase font-size and speed
func (s *Service) GetH5SettingHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"c": 0,
		"d": s.config.AllSettings(),
		"m": "",
	})
}
