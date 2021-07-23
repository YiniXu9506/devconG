package service

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
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

var token = "XVlBzgbaiCMRAjWwhTHctcuAxhxKQF"

// return phrases to wechat
func (s *Service) GetScrollingPhrasesHandler(c *gin.Context) {
	const defaultLimit = "100"
	limit, err := strconv.Atoi(c.DefaultQuery("limit", defaultLimit))

	if err != nil {
		fmt.Printf("failed to convert string to int")
		limit = 100
	}

	scrollingPhrasesRes := s.phraseCacheProvider.GetScrollingPhrases(limit)

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

	start := time.Now()

	if err := s.db.Table("phrase_models").
		Create(&model.PhraseModel{Text: req.Text, OpenID: req.OpenID, GroupID: req.GroupID, Status: 1, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}).Error; err != nil {
		mysqlErr := &mysql.MySQLError{}
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusBadRequest, gin.H{
				"c": 10001,
				"d": "",
				"m": "An existing item already exists",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
		}
		return
	}

	zap.L().Sugar().Infof("add new phrase cost: %v", time.Since(start))
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

	// start := time.Now()

	for _, phrase := range req {
		var phraseRecord model.PhraseModel
		phrase_id := phrase.PhraseID
		clicks := phrase.Clicks
		open_id := phrase.OpenID
		group_id := phrase.GroupID

		// check validation of phrase in phrase_models
		phraseRecordRe := s.db.Table("phrase_models").Where("phrase_id = ? AND status = ?", phrase_id, 2).Find(&phraseRecord)

		if phraseRecordRe.Error != nil {
			zap.L().Sugar().Error("Error! Check validation of phrase in phrase_models:", phraseRecordRe.Error)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": phraseRecordRe.Error.Error(),
			})
			return
		}

		// if find the reviewed phrase exist in phrase_models, then insert the click stats
		if phraseRecordRe.RowsAffected > 0 {
			if err := s.db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now().Unix()}).Error; err != nil {
				zap.L().Sugar().Error("Error! Failed to update phrase click model: ", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"c": 1,
					"d": "",
					"m": err.Error(),
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

func (s *Service) AddUserHandler(c *gin.Context) {
	var req model.UserModel
	// bind json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "open_id is required!",
		})
		return
	}

	if err := s.db.Table("user_models").
		Create(&model.UserModel{OpenID: req.OpenID, NickName: req.NickName, Sex: req.Sex, Province: req.Province, City: req.City, HeadImgURL: req.HeadImgURL}).Error; err != nil {
		mysqlErr := &mysql.MySQLError{}
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusOK, gin.H{
				"c": 0,
				"d": "",
				"m": "",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
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

// get all phrases
func (s *Service) GetAllPhrasesHandler(c *gin.Context) {
	reqToken := c.Request.Header.Get("token")

	if reqToken != token {
		c.JSON(http.StatusOK, gin.H{
			"c": -1,
			"d": "",
			"m": "invalid token",
		})

		return
	}

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

	start := time.Now()

	// get total counts of phrases
	if err := s.db.Table("phrase_models").
		Select("count(*)").
		Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", statusMap[0]), sql.Named("status2", statusMap[1]), sql.Named("status3", statusMap[2])).
		Find(&phraseTotalCount).Error; err != nil {
		zap.L().Sugar().Error("Error! Get total counts of phrases: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": err.Error(),
		})
		return
	}
	// get phrases with limit and offset
	if err := s.db.Table("phrase_models").
		Where("status = @status1 OR status = @status2 OR status = @status3", sql.Named("status1", statusMap[0]), sql.Named("status2", statusMap[1]), sql.Named("status3", statusMap[2])).
		Order("create_time desc").
		Limit(limit).
		Offset(offset).
		Find(&phraseList).Error; err != nil {
		zap.L().Sugar().Error("Error! Get phrases with limit and offset: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": err.Error(),
		})
		return
	}

	allPhrasesResp.Pagi.Total = phraseTotalCount
	allPhrasesResp.Pagi.Offset = offset

	for _, phrase := range phraseList {
		var phraseWithDistribution phraseWithDistributionModel

		// get all phrases from phrase_models
		if err := s.db.Table("phrase_click_models").
			Select("group_id, SUM(clicks) as clicks").
			Where("phrase_id = @phrase_id", sql.Named("phrase_id", phrase.PhraseID)).
			Group("group_id").
			Find(&distributions).Error; err != nil {
			zap.L().Sugar().Error("Error! Get all phrases from phrase_models: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}
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

	zap.L().Sugar().Infof("get all phrases cost: %v", time.Since(start))

	allPhrasesResp.List = allPhrasesWithDistributions

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": allPhrasesResp,
		"m": "",
	})
}

// get top-N phrases
func (s *Service) GetTopNPhrasesHandler(c *gin.Context) {
	reqToken := c.Request.Header.Get("token")

	if reqToken != token {
		c.JSON(http.StatusOK, gin.H{
			"c": -1,
			"d": "",
			"m": "invalid token",
		})

		return
	}

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

	start := time.Now()

	// get top N phrases, which are reviewed
	if err := s.db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_models as a INNER JOIN phrase_click_models as b ON a.phrase_id = b.phrase_id and a.status = 2 group by a.phrase_id order by clicks desc limit @limit", sql.Named("limit", limit)).
		Find(&topPhraseIDs).Error; err != nil {
		zap.L().Sugar().Error("Error! Get top N phrases, which are reviewed: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": err.Error(),
		})
		return
	}

	for _, phrase := range topPhraseIDs {
		var distributions []distributionModel
		var topPhraseText textModel
		var phraseWithDistribution topNPhrasesWithDistribution

		// get top N phrases from phrase_models
		if err := s.db.Table("phrase_click_models").
			Select("group_id, SUM(clicks) as clicks").
			Where("phrase_id = ?", phrase.PhraseID).
			Group("group_id").
			Find(&distributions).Error; err != nil {
			zap.L().Sugar().Error("Error! Get top N phrases from phrase_models: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}

		// get text of top N phrases from phrase_models
		if err := s.db.Table("phrase_models").
			Select("text").
			Where("phrase_id = ?", phrase.PhraseID).
			Find(&topPhraseText).Error; err != nil {
			zap.L().Sugar().Error("Error! Get text of top N phrases from phrase_models: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}

		phraseWithDistribution.PhraseID = phrase.PhraseID
		phraseWithDistribution.Text = topPhraseText.Text
		phraseWithDistribution.Distributions = phraseDistribution(distributions)

		topNPhrasesWithDistributions = append(topNPhrasesWithDistributions, phraseWithDistribution)
	}

	zap.L().Sugar().Infof("get top phrase cost: %v", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": topNPhrasesWithDistributions,
		"m": "",
	})
}

// delete phrase by change status to 3
func (s *Service) DeletePhraseHandler(c *gin.Context) {
	reqToken := c.Request.Header.Get("token")

	if reqToken != token {
		c.JSON(http.StatusOK, gin.H{
			"c": -1,
			"d": "",
			"m": "invalid token",
		})

		return
	}

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

	start := time.Now()

	deletePhraseRes := s.db.Table("phrase_models").
		Where("phrase_id = ?", req.PhraseID).
		Updates(map[string]interface{}{"status": 3, "update_time": time.Now().Unix()})

	if deletePhraseRes.Error != nil {
		zap.L().Sugar().Error("Error! Delete phrase: ", deletePhraseRes.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": deletePhraseRes.Error.Error(),
		})
		return
	}

	if deletePhraseRes.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 11001,
			"d": "",
			"m": "Nonexistent",
		})
		return
	}

	zap.L().Sugar().Infof("delete phrase cost: %v", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// update phrase text or status
func (s *Service) PatchPhraseHandler(c *gin.Context) {
	reqToken := c.Request.Header.Get("token")

	if reqToken != token {
		c.JSON(http.StatusOK, gin.H{
			"c": -1,
			"d": "",
			"m": "invalid token",
		})

		return
	}

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

	// check whether the phrase exist or not
	phraseRes := s.db.Table("phrase_models").Where("phrase_id = ?", req.PhraseID).Find(&row)
	if phraseRes.Error != nil {
		zap.L().Sugar().Error("Error! Get phrase to update its text or status", phraseRes.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": phraseRes.Error.Error(),
		})
		return
	}

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
		if err := s.db.Table("phrase_models").
			Where("phrase_id = ?", req.PhraseID).
			Updates(updates).Error; err != nil {
			zap.L().Sugar().Error("Error! Update phrase text or status", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// batch update reviewed phrase
func (s *Service) PatchBatchPhraseHandler(c *gin.Context) {
	reqToken := c.Request.Header.Get("token")

	if reqToken != token {
		c.JSON(http.StatusOK, gin.H{
			"c": -1,
			"d": "",
			"m": "invalid token",
		})

		return
	}

	type batchReviewPhraseReq struct {
		PhraseID []int `form:"ids" json:"ids" binding:"required"`
		Status   int   `form:"status" json:"status" binding:"required"`
	}

	var req batchReviewPhraseReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"c": 2,
			"d": "",
			"m": "phrase_id and status are required!",
		})
		return
	}

	// batch review phrase
	selectPhrasesWithStatus := 1
	updateStatusTo := 2

	// batch delete pharse
	if req.Status == 3 {
		selectPhrasesWithStatus = 2
		updateStatusTo = 3
	}

	if req.Status == 2 || req.Status == 3 {
		if err := s.db.Table("phrase_models").Where("status = ? AND phrase_id IN ?", selectPhrasesWithStatus, req.PhraseID).Updates(map[string]interface{}{"status": updateStatusTo, "update_time": time.Now().Unix()}).Error; err != nil {
			zap.L().Sugar().Error("Error! Update phrase text or status", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

// get phrase font-size and speed
func (s *Service) GetH5SettingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": s.config.AllSettings(),
		"m": "",
	})
}

func (s *Service) TestPhrasePostHandler(c *gin.Context) {

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	str := make([]byte, 10)
	for i := range str {
		str[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	type phrase struct {
		Text    string `json:"text"`
		OpenID  string `json:"open_id"`
		GroupID int    `json:"group_id"`
	}

	newPhrase := phrase{
		Text:    string(str),
		GroupID: rand.Intn(5) + 1,
		OpenID:  fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
	}

	// start := time.Now()

	if err := s.db.Table("phrase_models").
		Create(&model.PhraseModel{Text: newPhrase.Text, OpenID: newPhrase.OpenID, GroupID: newPhrase.GroupID, Status: rand.Intn(3) + 1, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}).Error; err != nil {
		mysqlErr := &mysql.MySQLError{}
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusBadRequest, gin.H{
				"c": 10001,
				"d": "",
				"m": "An existing item already exists",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
		}
		return
	}

	// zap.L().Sugar().Infof("test add new phrase cost: %v", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

func (s *Service) TestPhraseHotPostHandler(c *gin.Context) {
	type phraseClick struct {
		PhraseID int    `json:"phrase_id"`
		Clicks   int    `json:"clicks"`
		OpenID   string `json:"open_id"`
		GroupID  int    `json:"group_id"`
	}

	newPhraseClick := phraseClick{
		PhraseID: rand.Intn(10000000) + 1,
		GroupID:  rand.Intn(5) + 1,
		OpenID:   fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
		Clicks:   rand.Intn(5) + 1,
	}

	var phraseRecord model.PhraseModel
	phrase_id := newPhraseClick.PhraseID
	clicks := newPhraseClick.Clicks
	open_id := newPhraseClick.OpenID
	group_id := newPhraseClick.GroupID

	// start := time.Now()

	// check validation of phrase in phrase_models
	phraseRecordRe := s.db.Table("phrase_models").Where("phrase_id = ? AND status = ?", phrase_id, 2).Find(&phraseRecord)

	if phraseRecordRe.Error != nil {
		zap.L().Sugar().Error("Error! Check validation of phrase in phrase_models:", phraseRecordRe.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"c": 1,
			"d": "",
			"m": phraseRecordRe.Error.Error(),
		})
		return
	}

	// if find the reviewed phrase exist in phrase_models, then insert the click stats
	if phraseRecordRe.RowsAffected > 0 {
		if err := s.db.Create(&model.PhraseClickModel{PhraseID: phrase_id, Clicks: clicks, OpenID: open_id, GroupID: group_id, ClickTime: time.Now().Unix()}).Error; err != nil {
			zap.L().Sugar().Error("Error! Failed to update phrase click model: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
			})
			return
		}
	}

	// zap.L().Sugar().Infof("Test update phrase click cost: %v", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": "",
		"m": "",
	})
}

func (s *Service) TestUserPostHandler(c *gin.Context) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	str := make([]byte, 10)
	for i := range str {
		str[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	req := model.UserModel{
		OpenID:     string(str),
		NickName:   string(str),
		Sex:        rand.Intn(2) + 1,
		Province:   "广州",
		City:       string(str),
		HeadImgURL: string(str),
	}

	if err := s.db.Table("user_models").
		Create(&model.UserModel{OpenID: req.OpenID, NickName: req.NickName, Sex: req.Sex, Province: req.Province, City: req.City, HeadImgURL: req.HeadImgURL}).Error; err != nil {
		mysqlErr := &mysql.MySQLError{}
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			c.JSON(http.StatusOK, gin.H{
				"c": 0,
				"d": "",
				"m": "",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"c": 1,
				"d": "",
				"m": err.Error(),
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

func (s *Service) GetOverviewHandler(c *gin.Context) {
	// sex stats
	type sexModel struct {
		Sex   int `json:"sex"`
		Count int `json:"count"`
	}
	var sexRecords []sexModel
	if err := s.db.Table("user_models").
		Select("sex, count(*) as count").
		Group("sex").
		Find(&sexRecords).Error; err != nil {
		zap.L().Sugar().Error("Error! Get user distrbution failed: ", err)
		return
	}
	type sexResponseModel struct {
		Male   int `json:"male"`
		Female int `json:"female"`
		Secret int `json:"secret"`
	}
	var sexRes sexResponseModel
	var totalUser int
	for _, r := range sexRecords {
		switch r.Sex {
		case 1:
			sexRes.Male = r.Count
		case 2:
			sexRes.Female = r.Count
		case 3:
			sexRes.Secret = r.Count
		}

		totalUser += r.Count
	}

	// location stats
	type locationModel struct {
		Province string `json:"province"`
		Count    int    `json:"count"`
	}
	var locationsRecords []locationModel

	start := time.Now()

	if err := s.db.Table("user_models").
		Select("province, count(*) as count").
		Group("province").
		Order("count desc, province desc").
		Limit(5).
		Find(&locationsRecords).Error; err != nil {
		zap.L().Sugar().Error("Error! Get user distrbution failed: ", err)
		return
	}

	var top5LocationsCount int
	for _, location := range locationsRecords {
		top5LocationsCount += location.Count
	}

	if top5LocationsCount > 0 {
		otherLocations := locationModel{
			Province: "其他",
			Count:    totalUser - top5LocationsCount,
		}

		locationsRecords = append(locationsRecords, otherLocations)
	}

	type responseModel struct {
		TotalUser        int              `json:"total_users"`
		TotalValidPhrase int              `json:"total_valid_phrases"`
		TotalClicks      int              `json:"total_clicks"`
		Sex              sexResponseModel `json:"sex"`
		Localtions       []locationModel  `json:"locations"`
	}

	var totalValidPhrase int
	if err := s.db.Table("phrase_models").
		Select("count(*)").
		Where("status = ?", 2).
		Find(&totalValidPhrase).Error; err != nil {
		zap.L().Sugar().Error("Error! Get total valid phrase failed: ", err)
		return
	}

	type sumClickModel struct {
		Clicks int `json:"clicks"`
	}

	var totalClicks []sumClickModel
	if err := s.db.Table("phrase_click_models").
		Select("sum(clicks) as clicks").
		Scan(&totalClicks).Error; err != nil {
		zap.L().Sugar().Error("Error! Get total clicks failed: ", err)
		return
	}

	zap.L().Sugar().Infof("get overview cost: %v", time.Since(start))
	var resp responseModel

	resp.TotalUser = totalUser

	resp.TotalClicks = totalClicks[0].Clicks
	resp.TotalValidPhrase = totalValidPhrase
	resp.Sex = sexRes
	resp.Localtions = locationsRecords

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": resp,
		"m": "",
	})
}

func (s *Service) GetClickTrendsHandler(c *gin.Context) {
	type clickTrendsModel struct {
		Time   int64 `json:"time"`
		Clicks int   `json:"clicks"`
	}

	var clickTrendsRecords, clickTrendsResp []clickTrendsModel

	start := time.Now()

	if err := s.db.Raw("select time, agg_clicks as clicks from (SELECT *, sum(clicks) over (partition by gid order by time) as agg_clicks from  (SELECT 1 as gid, ceiling(click_time/600)*600 as time, sum(clicks) as clicks FROM phrase_click_models GROUP BY  ceiling(click_time/600)) as t ) as tt WHERE tt.time > UNIX_TIMESTAMP(NOW() - INTERVAL 3 HOUR);").Scan(&clickTrendsRecords).Error; err != nil {
		zap.L().Sugar().Error("Error! Get click trends failed: ", err)
		return
	}

	t := time.Now().Add(-time.Duration(170) * time.Minute)

	timeArr := make([]int64, 18)

	for i := 0; i < 18; i++ {
		timeArr[i] = t.Add(time.Duration(10*(i+1))*time.Minute).Unix() / 600 * 600
	}

	for i, t := range timeArr {
		trend := clickTrendsModel{
			Time:   t,
			Clicks: 0,
		}
		if i != 0 {
			trend.Clicks = clickTrendsResp[i-1].Clicks
		} else {
			var total int
			s.db.Raw(fmt.Sprintf("select sum(clicks) from phrase_click_models where click_time < %d", t)).First(&total)
			trend.Clicks = total
		}

		for _, resp := range clickTrendsRecords {
			if resp.Time == t {
				trend.Clicks = resp.Clicks
				break
			}
		}

		clickTrendsResp = append(clickTrendsResp, trend)
	}

	zap.L().Sugar().Infof("get click trends cost: %v", time.Since(start))

	c.JSON(http.StatusOK, gin.H{
		"c": 0,
		"d": clickTrendsResp,
		"m": "",
	})
}
