package provider

import (
	"database/sql"
	"math/rand"
	"sync"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrollingPhrasesResponse struct {
	PhraseID       int    `json:"phrase_id"`
	Text           string `json:"text"`
	Clicks         int    `json:"clicks"`
	HotGroupID     int    `json:"hot_group_id"`
	HotGroupClicks int    `json:"hot_group_clicks"`
}

type PhrasesCacheProvider struct {
	db            *gorm.DB
	cachedPhrases []ScrollingPhrasesResponse
	mu            sync.RWMutex
}

func NewPhrasesCacheProvider(db *gorm.DB) *PhrasesCacheProvider {
	phraseCache := &PhrasesCacheProvider{
		db:            db,
		cachedPhrases: make([]ScrollingPhrasesResponse, 0, 100),
	}
	go periodUpdateCache(phraseCache)
	return phraseCache
}

/* if the counts of reviewed phrase are less than the limit, set the limit to reviewedPhraseCount
calculate and update phrases:
append 30% neweset phrases, whose status need to be reviewd
append 30% hot phrases
append 40% random phrases
*/
func getReturnPhraseCount(limit int, reviewedPhraseCount int, db *gorm.DB) (int, int, int) {
	if reviewedPhraseCount < limit {
		limit = reviewedPhraseCount
	}

	newestPhrasesCount := int(float64(limit) * 0.3)
	topNPhrasesCount := int(float64(limit) * 0.3)

	return newestPhrasesCount, topNPhrasesCount, limit
}

// get scrolling phrase from phraseCache according to limit
func (cp *PhrasesCacheProvider) GetScrollingPhrases(limit int) []ScrollingPhrasesResponse {
	cp.mu.RLock()

	var phrase []ScrollingPhrasesResponse

	reviewedPhraseCount := len(cp.cachedPhrases)
	newestPhrasesCount, topNPhrasesCount, limit := getReturnPhraseCount(limit, reviewedPhraseCount, cp.db)
	randeomPhraseCount := limit - newestPhrasesCount - topNPhrasesCount

	// sliceGap uses to slice phrase
	sliceGap := int(float64(reviewedPhraseCount) * 0.3)

	phrase = append(phrase, cp.cachedPhrases[:newestPhrasesCount]...)
	phrase = append(phrase, cp.cachedPhrases[sliceGap:sliceGap+topNPhrasesCount]...)
	phrase = append(phrase, cp.cachedPhrases[2*sliceGap:2*sliceGap+randeomPhraseCount]...)

	rand.Shuffle(len(phrase), func(i, j int) {
		phrase[i], phrase[j] = phrase[j], phrase[i]
	})

	defer cp.mu.RUnlock()

	return phrase
}

func (cp *PhrasesCacheProvider) updateCache() {

	type topClicksPhraseModel struct {
		Clicks   int `json:"clicks"`
		PhraseID int `json:"phrase_id"`
		GroupID  int `json:"group_id"`
	}

	var scrollingPhrasesRes []ScrollingPhrasesResponse
	var newestPhrases, randomPickPhrases []model.PhraseModel
	var topClicksPhrases []topClicksPhraseModel

	start := time.Now()

	var reviewedPhraseCount int
	limit := 100
	if err := cp.db.Raw("Select count(*) from phrase_models where status=2").
		Find(&reviewedPhraseCount).Error; err != nil {
		zap.L().Sugar().Error("Error! Select reviewed phrases counts: ", err)
		return
	}

	newestPhrasesCount, topNPhrasesCount, limit := getReturnPhraseCount(limit, reviewedPhraseCount, cp.db)

	// get newest-N phrases
	if err := cp.db.Table("phrase_models").
		Where("status = ?", 2).
		Order("update_time desc").
		Limit(newestPhrasesCount).
		Find(&newestPhrases).Error; err != nil {
		zap.L().Sugar().Error("Error! Get newest-N phrases: ", err)
		return
	}

	// get top-N click phrases
	if err := cp.db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_click_models as a LEFT JOIN phrase_models as b ON a.phrase_id = b.phrase_id and b.status = 2 group by a.phrase_id order by clicks desc limit @limit", sql.Named("limit", topNPhrasesCount)).
		Scan(&topClicksPhrases).Error; err != nil {
		zap.L().Sugar().Error("Error! Get top N clicks phrases: ", err)
		return
	}

	// de-duplicate phrase
	allIDs := make(map[int]bool)
	var allIDSorted []int
	for _, item := range newestPhrases {
		if res, ok := allIDs[item.PhraseID]; !ok || !res {
			allIDSorted = append(allIDSorted, item.PhraseID)
		}

		allIDs[item.PhraseID] = true
	}
	for _, item := range topClicksPhrases {
		if res, ok := allIDs[item.PhraseID]; !ok || !res {
			allIDSorted = append(allIDSorted, item.PhraseID)
		}
		allIDs[item.PhraseID] = true
	}

	// get more random phrase if de-duplicate topNPhrases and newestPhrases
	for len(allIDs) < limit {
		delta := limit - len(allIDs)
		if err := cp.db.Raw("SELECT * FROM phrase_models where status = 2 ORDER BY RAND() LIMIT ?", delta).
			Scan(&randomPickPhrases).Error; err != nil {
			zap.L().Sugar().Error("Error! Get random phrases: ", err)
			return
		}

		for _, item := range randomPickPhrases {
			if res, ok := allIDs[item.PhraseID]; !ok || !res {
				allIDSorted = append(allIDSorted, item.PhraseID)
			}
			allIDs[item.PhraseID] = true
		}
	}

	for _, id := range allIDSorted {
		var phrase ScrollingPhrasesResponse
		var phraseRecord model.PhraseModel
		var topClickPhrases topClicksPhraseModel
		var topClickGroup topClicksPhraseModel

		type phraseUpdateTimeModel struct {
			ClickTime int64 `json:"click_time"`
		}
		var phraseUpdateTime phraseUpdateTimeModel

		if err := cp.db.Table("phrase_models").
			Select("phrase_id, text").
			Where("phrase_id = ?", id).
			Find(&phrase).Error; err != nil {
			zap.L().Sugar().Error("Error! Retrive phrase from db: ", err)
			return
		}

		// find out which group has top clicks on specific phrase
		//TODO: error handling
		if err := cp.db.Table("phrase_click_models").
			Select("sum(clicks) as clicks, phrase_id, group_id").
			Where("phrase_id = ?", id).
			Group("phrase_id, group_id").
			Order("clicks desc").
			Limit(1).
			Find(&topClickGroup).Error; err != nil {
			zap.L().Sugar().Error("Error! Retrive top clicks group: ", err)
			return
		}

		if topClickGroup.GroupID == 0 {
			// if this phrase has not been clicked, the group_id will be the poster belongs to.
			if err := cp.db.Table("phrase_models").
				Select("group_id").Where("phrase_id = ?", id).
				Find(&phraseRecord).Error; err != nil {
				zap.L().Sugar().Error("Error! Get the groupID if the phrase has not been clicked: ", err)
				return
			}
			topClickGroup.GroupID = phraseRecord.GroupID
		} else {
			// find out all click counts of a specific phrase
			if err := cp.db.Table("phrase_click_models").
				Select("sum(clicks) as clicks, phrase_id").
				Where("phrase_id = ?", id).
				Group("phrase_id").
				Find(&topClickPhrases).Error; err != nil {
				zap.L().Sugar().Error("Error! Find out all click counts of a specific phrase: ", err)
				return
			}

			// update phrase click time
			if err := cp.db.Table("phrase_click_models").
				Select("click_time").
				Where("phrase_id = ?", id).
				Order("click_time desc").
				Limit(1).
				Find(&phraseUpdateTime).Error; err != nil {
				zap.L().Sugar().Error("Error! Get the newest click time of phrase: ", err)
				return
			}
			if err := cp.db.Table("phrase_models").
				Where("phrase_id = ?", id).
				Update("update_time", phraseUpdateTime.ClickTime).Error; err != nil {
				zap.L().Sugar().Error("Error! Update phrase interactive time: ", err)
				return
			}
		}

		phrase.Clicks = topClickPhrases.Clicks
		phrase.HotGroupClicks = topClickGroup.Clicks
		phrase.HotGroupID = topClickGroup.GroupID
		scrollingPhrasesRes = append(scrollingPhrasesRes, phrase)
	}

	zap.L().Sugar().Infof("update phrase cache cost: %v", time.Since(start))

	cp.mu.Lock()
	cp.cachedPhrases = scrollingPhrasesRes
	cp.mu.Unlock()
}

func periodUpdateCache(cache *PhrasesCacheProvider) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		<-ticker.C
		cache.updateCache()
	}
}
