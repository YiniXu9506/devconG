package service

import (
	"github.com/YiniXu9506/devconG/provider"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Service struct {
	db                       *gorm.DB
	phraseCacheProvider      *provider.PhrasesCacheProvider
	clickTrendsCacheProvider *provider.ClickTrendsCacheProvider
	config                   *viper.Viper
}

func NewService(db *gorm.DB, config *viper.Viper) *Service {
	phraseCacheProvider := provider.NewPhrasesCacheProvider(db)
	clickTrendsCacheProvider := provider.NewClickTrendsCacheProvider(db)

	return &Service{
		db:                       db,
		phraseCacheProvider:      phraseCacheProvider,
		clickTrendsCacheProvider: clickTrendsCacheProvider,
		config:                   config,
	}
}

func (s *Service) Start(r *gin.Engine) {
	// APIs for wechat mini program
	r.GET("/phrases", s.GetScrollingPhrasesHandler)
	r.POST("/phrase", s.AddPhraseHandler)
	r.POST("/phrase_hot", s.UpdateClickedPhraseHandler)
	r.POST("/user", s.AddUserHandler)
	r.GET("/h5_settings", s.GetH5SettingHandler)
	r.GET("/test-phrase-post", s.TestPhrasePostHandler)
	r.GET("/test-phrase-hot-post", s.TestPhraseHotPostHandler)
	r.GET("/test-user-post", s.TestUserPostHandler)

	// APIs for management portal
	r.GET("/phrases_full", s.GetAllPhrasesHandler)
	r.GET("/top_phrases", s.GetTopNPhrasesHandler)
	r.DELETE("/phrase", s.DeletePhraseHandler)
	r.PATCH("/phrase", s.PatchPhraseHandler)

	// API for BI
	r.GET("/overview", s.GetOverviewHandler)
	r.GET("/click_trends", s.GetClickTrendsHandler)
}
