package utils

import (
	"../mock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TiDBConnect() *gorm.DB {
	dsn := "root@tcp(127.0.0.1:4000)/test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&mock.PhraseClickModel{}, &mock.PhraseModel{})

	// mock.MockPhraseClick(100, db)

	// mock.MockPhrase(50, db)

	return db
}
