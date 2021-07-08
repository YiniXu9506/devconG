package main

import (
	"flag"
	"fmt"

	"github.com/YiniXu9506/devconG/api"
	"github.com/YiniXu9506/devconG/model"
	"github.com/YiniXu9506/devconG/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var config *viper.Viper

func initConfigure(configFileName string) *viper.Viper {
	v := viper.New()
	fmt.Printf("filaname %v\n", configFileName)

	v.SetConfigName(configFileName) // name of config file (without extension)
	v.SetConfigType("json")         // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath("./")           // path to look for the config file in
	v.Set("verbose", true)          // 设置默认参数

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(" Config file not found; ignore error if desired")
		} else {
			panic("Config file was found but another error was produced")
		}
	}
	// viper runs each time a change occurs.
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	return v
}

func main() {
	var configFileName = flag.String("f", "config", "customize the filename")
	config = initConfigure(*configFileName)
	viper.SetDefault("ContentDir", "content")
	viper.SetDefault("LayoutDir", "layouts")
	viper.SetDefault("Taxonomies", map[string]string{"tag": "tags", "category": "categories"})

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

	r.GET("/phrases_full", func(c *gin.Context) {
		api.GetAllPhrases(c, db)
	})

	r.GET("/top_phrases", func(c *gin.Context) {
		api.GetTopNPhrases(c, db)
	})

	// r.DELETE("/phrase", func(c *gin.Context) {
	// 	api.DeletePhrase(c, db)
	// })

	r.GET("/h5_settings", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"c": 0,
			"d": config.AllSettings(),
			"m": "",
		})
	})

	r.Run()
}
