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

type TopClicksPhraseModel struct {
	Clicks   int `json:"clicks"`
	PhraseID int `json:"phrase_id"`
	GroupID  int `json:"group_id"`
}

type ClickTrendsResponse struct {
	Time   string `json:"time"`
	Clicks int    `json:"clicks"`
}

type ClickTrendsCacheProvider struct {
	db                *gorm.DB
	cachedClickTrends []ClickTrendsResponse
	mu                sync.RWMutex
}

func NewClickTrendsCacheProvider(db *gorm.DB) *ClickTrendsCacheProvider {
	clickTrendsCache := &ClickTrendsCacheProvider{
		db:                db,
		cachedClickTrends: make([]ClickTrendsResponse, 0, 100),
	}

	go periodUpdateClickTrendsCache(clickTrendsCache)

	return clickTrendsCache
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

func CacheNPhrases(id int, cp *PhrasesCacheProvider, c chan ScrollingPhrasesResponse) {
	var phrase ScrollingPhrasesResponse
	var phraseRecord model.PhraseModel
	var phraseClicksDistribution []TopClicksPhraseModel
	var topClickGroup TopClicksPhraseModel
	totalClicks := 0
	defer func() {
		c <- phrase
	}()
	if err := cp.db.Table("phrase_models").
		Select("phrase_id, text, group_id").
		Where("phrase_id = ?", id).
		Find(&phraseRecord).Error; err != nil {
		zap.L().Sugar().Error("Error! Retrive phrase from db: ", err)
		return
	}

	// find out phrase click distributions
	if err := cp.db.Table("phrase_click_models").
		Select("sum(clicks) as clicks, phrase_id, group_id").
		Where("phrase_id = ?", id).
		Group("phrase_id, group_id").
		Order("clicks").
		Find(&phraseClicksDistribution).Error; err != nil {
		zap.L().Sugar().Error("Error! Retrive top clicks group: ", err)
		return
	}

	if len(phraseClicksDistribution) == 0 {
		topClickGroup.GroupID = phraseRecord.GroupID
		totalClicks = 0
	} else {
		// sum up clicks for phrase
		for _, distribution := range phraseClicksDistribution {
			totalClicks = totalClicks + distribution.Clicks
		}

		topClickGroup.Clicks = phraseClicksDistribution[0].Clicks
		topClickGroup.GroupID = phraseClicksDistribution[0].GroupID

		// update phrase show time
		if err := cp.db.Table("phrase_models").
			Where("phrase_id = ?", id).
			Update("update_time", time.Now().Unix()).Error; err != nil {
			zap.L().Sugar().Error("Error! Update phrase interactive time: ", err)
			return
		}
	}

	phrase.PhraseID = phraseRecord.PhraseID
	phrase.Text = phraseRecord.Text
	phrase.Clicks = totalClicks
	phrase.HotGroupClicks = topClickGroup.Clicks
	phrase.HotGroupID = topClickGroup.GroupID

}

func getNewestNPhrase(db *gorm.DB, newestPhrasesCount int, c chan []model.PhraseModel) {
	var newestPhrases []model.PhraseModel

	defer func() {
		c <- newestPhrases
	}()

	// get newest-N phrases
	if err := db.Table("phrase_models").
		Where("status = ?", 2).
		Order("update_time").
		Limit(newestPhrasesCount).
		Find(&newestPhrases).Error; err != nil {
		zap.L().Sugar().Error("Error! Get newest-N phrases: ", err)
		return
	}

}

func getTopNPhrase(db *gorm.DB, topNPhrasesCount int, c chan []TopClicksPhraseModel) {
	var topClicksPhrases []TopClicksPhraseModel
	defer func() {
		c <- topClicksPhrases
	}()

	if err := db.Raw("SELECT sum(clicks) as clicks, a.phrase_id FROM phrase_click_models as a LEFT JOIN phrase_models as b ON a.phrase_id = b.phrase_id and b.status = 2 group by a.phrase_id order by clicks desc limit @limit", sql.Named("limit", topNPhrasesCount)).
		Scan(&topClicksPhrases).Error; err != nil {
		zap.L().Sugar().Error("Error! Get top N clicks phrases: ", err)
		return
	}
}

func getRandomNPhrase(db *gorm.DB, randomNPhrasesCount int, c chan []model.PhraseModel) {
	var randomPickPhrases []model.PhraseModel
	defer func() {
		c <- randomPickPhrases
	}()

	if err := db.Raw("SELECT * FROM phrase_models where status = 2 ORDER BY RAND() LIMIT ?", randomNPhrasesCount).
		Scan(&randomPickPhrases).Error; err != nil {
		zap.L().Sugar().Error("Error! Get random phrases: ", err)
		return
	}
}

func (cp *PhrasesCacheProvider) updateCache() {

	c := make(chan ScrollingPhrasesResponse)
	newestNPhraseC := make(chan []model.PhraseModel)
	topNPhraseC := make(chan []TopClicksPhraseModel)
	randomPhraseC := make(chan []model.PhraseModel)

	var newestPhrases, randomPickPhrases []model.PhraseModel
	var topClicksPhrases []TopClicksPhraseModel

	start := time.Now()

	var reviewedPhraseCount int
	limit := 30

	if err := cp.db.Raw("Select count(*) from phrase_models where status=2").
		Find(&reviewedPhraseCount).Error; err != nil {
		zap.L().Sugar().Error("Error! Select reviewed phrases counts: ", err)
		return
	}

	newestPhrasesCount, topNPhrasesCount, limit := getReturnPhraseCount(limit, reviewedPhraseCount, cp.db)

	// get newest-N phrases
	go getNewestNPhrase(cp.db, newestPhrasesCount, newestNPhraseC)

	// get top-N click phrases
	go getTopNPhrase(cp.db, topNPhrasesCount, topNPhraseC)

	// get more random phrase
	go getRandomNPhrase(cp.db, (limit-topNPhrasesCount-newestPhrasesCount)*4, randomPhraseC)

	newestPhrases = <-newestNPhraseC
	topClicksPhrases = <-topNPhraseC
	randomPickPhrases = <-randomPhraseC

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
	for _, item := range randomPickPhrases {
		if res, ok := allIDs[item.PhraseID]; !ok || !res {
			allIDSorted = append(allIDSorted, item.PhraseID)
		}
		allIDs[item.PhraseID] = true
	}

	for _, id := range allIDSorted {
		go CacheNPhrases(id, cp, c)
	}

	var phrases []ScrollingPhrasesResponse
	for _ = range allIDSorted {
		phrase := <-c
		if phrase.PhraseID == 0 && len(phrase.Text) == 0 {
			continue
		}
		phrases = append(phrases, phrase)
	}
	zap.L().Sugar().Infof("update phrase cache cost: %v", time.Since(start))
	cp.mu.Lock()
	cp.cachedPhrases = phrases
	cp.mu.Unlock()
}

func (ct *ClickTrendsCacheProvider) GetClickTrends() []ClickTrendsResponse {
	return ct.cachedClickTrends
}

func periodUpdateCache(cache *PhrasesCacheProvider) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		<-ticker.C
		cache.updateCache()
	}
}

func (ct *ClickTrendsCacheProvider) updateCache() {
	var clickTrendsRecords []ClickTrendsResponse
	start := time.Now()
	if err := ct.db.Debug().Raw("SELECT FROM_UNIXTIME(floor(click_time/600)*600, '%T') as time, sum(clicks) as clicks FROM phrase_click_models WHERE click_time > UNIX_TIMESTAMP(NOW() - INTERVAL 3 HOUR) GROUP BY floor(click_time/600) order by time;").Scan(&clickTrendsRecords).Error; err != nil {
		zap.L().Sugar().Error("Error! Get click trends failed: ", err)
		return
	}

	zap.L().Sugar().Infof("update click trends cache cost: %v", time.Since(start))

	ct.mu.Lock()
	ct.cachedClickTrends = clickTrendsRecords
	ct.mu.Unlock()
}

func periodUpdateClickTrendsCache(cache *ClickTrendsCacheProvider) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		<-ticker.C
		cache.updateCache()
	}
}
