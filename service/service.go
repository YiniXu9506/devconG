package service

import (
	"github.com/YiniXu9506/devconG/provider"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	provider *provider.PhrasesCacheProvider
	config   *viper.Viper
}

func NewService(db *gorm.DB, config *viper.Viper) *Service {
	provider := provider.NewPhrasesCacheProvider(db)
	return &Service{
		db:       db,
		provider: provider,
		config:   config,
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

	// APIs for management portal
	r.GET("/phrases_full", s.GetAllPhrasesHandler)
	r.GET("/top_phrases", s.GetTopNPhrasesHandler)
	r.DELETE("/phrase", s.DeletePhraseHandler)
	r.PATCH("/phrase", s.PatchPhraseHandler)
}
