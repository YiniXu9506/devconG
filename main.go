package main

import (
	"github.com/YiniXu9506/devconG/api"
	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	db := utils.TiDBConnect()

	cachePhrases := &model.CachePhrases{
		PhraseList: make([]model.PhraseItem, 0, 100),
	}

	go model.UpdateStats(db, cachePhrases)
	r.GET("/phrases", func(c *gin.Context) {
		api.GetSrollingPhrases(c, db, cachePhrases)
	})

	r.POST("/phrase", func(c *gin.Context) {
		api.AddPhrase(c, db)
	})

	r.POST("/phrase_hot", func(c *gin.Context) {
		api.UpdateClickedPhrase(c, db)
	})

	// r.GET("/phrases_full", func(c *gin.Context) {
	// 	api.GetAllPhrases(c, db)
	// })

	r.Run()
}
