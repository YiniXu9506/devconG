package main

import (
	"github.com/YiniXu9506/devconG/api"
	"github.com/YiniXu9506/devconG/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	db := utils.TiDBConnect()

	r.GET("/phrases", func(c *gin.Context) {
		api.GetPhrases(c, db)
	})

	r.POST("/phrase", func(c *gin.Context) {
		api.AddPhrase(c, db)
	})

	r.POST("/phrase_hot", func(c *gin.Context) {
		api.UpdateClickedPhrase(c, db)
	})

	r.Run()
}
