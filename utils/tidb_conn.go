package utils

import (
	"fmt"

	"github.com/YiniXu9506/devconG/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TiDBConnect(hostName string, port int) *gorm.DB {
	dsn := fmt.Sprintf("root@tcp(%v:%v)/test?charset=utf8mb4&parseTime=True&loc=Local", hostName, port)
	fmt.Printf("dsn %v\n", dsn)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&model.PhraseClickModel{}, &model.PhraseModel{})

	// model.MockPhraseClick(50, db)

	// model.MockPhrase(50, db)

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
