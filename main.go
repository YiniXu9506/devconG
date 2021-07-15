package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/YiniXu9506/devconG/log"
	"github.com/YiniXu9506/devconG/service"
	"github.com/YiniXu9506/devconG/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var config *viper.Viper
var configFileName = flag.String("f", "config", "customize the filename.")
var hostName = flag.String("h", "127.0.0.1", "Connect to host.")
var port = flag.Int("P", 4000, "the database ports.")
var serverPort = flag.Int("l", 8080, "Port number listenling.")

func initConfigure(configFileName string) *viper.Viper {
	v := viper.New()

	v.SetConfigName(configFileName) // name of config file (without extension)
	v.SetConfigType("json")         // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath("./")           // path to look for the config file in

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

func init() {
	// initial log
	log.SetLogs(zap.ErrorLevel, log.LOGFORMAT_CONSOLE, "./server.log")
}

func main() {
	// TODO: add flags for database connection and period to
	flag.Parse()
	config = initConfigure(*configFileName)

	r := gin.New()
	pprof.Register(r)
	r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))

	db := utils.TiDBConnect(*hostName, *port)
	service := service.NewService(db, config)
	service.Start(r)

	r.Use(cors.Default())

	r.Run(fmt.Sprintf(":%d", *serverPort))
}
