package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gorm.io/gorm/logger"
)

func TiDBConnect(hostName string, port int) *gorm.DB {
	dsn := fmt.Sprintf("root@tcp(%v:%v)/test?charset=utf8mb4&parseTime=True&loc=Local", hostName, port)
	fmt.Printf("dsn %v\n", dsn)
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             6 * time.Second, // Slow SQL threshold
			LogLevel:                  logger.Silent,   // Log level
			IgnoreRecordNotFoundError: true,            // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,           // Disable color
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.PhraseClickModel{}, &model.PhraseModel{}, &model.UserModel{})

	model.MockPhraseClick(50, db)

	model.MockPhrase(50, db)

	return db
}

// MySQLError is an error type which represents a single MySQL error
type MySQLError struct {
	Number  uint16
	Message string
}

func (me MySQLError) Error() string {
	return fmt.Sprintf("Error %d: %s", me.Number, me.Message)
}
