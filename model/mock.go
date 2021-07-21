package model

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MockPhraseClick(n int, db *gorm.DB) {
	t := time.Now().Add(-time.Duration(n) * time.Minute)
	for i := 1; i <= n; i++ {
		phraseClick := PhraseClickModel{
			ID:        i,
			GroupID:   rand.Intn(5) + 1,
			OpenID:    fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
			PhraseID:  rand.Intn(50) + 1,
			Clicks:    rand.Intn(5) + 1,
			ClickTime: t.Add(time.Duration(i) * time.Minute).Unix(),
		}

		db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&phraseClick)
	}
}

func MockPhrase(n int, db *gorm.DB) {
	for i := 1; i <= n; i++ {
		phrase := PhraseModel{
			PhraseID:   i,
			Text:       fmt.Sprintf("tidb%v", i),
			GroupID:    rand.Intn(5) + 1,
			OpenID:     fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
			Status:     rand.Intn(3) + 1,
			CreateTime: time.Now().Unix(),
			UpdateTime: time.Now().Unix(),
		}

		db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&phrase)
	}
}

func MockUser(n int, db *gorm.DB) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i := 1; i <= n; i++ {
		str := make([]byte, 10)
		for i := range str {
			str[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		user := UserModel{
			OpenID:     string(str),
			NickName:   string(str),
			Sex:        rand.Intn(2) + 1,
			Province:   fmt.Sprintf("province%d", rand.Intn(10)+1),
			City:       string(str),
			HeadImgURL: string(str),
		}

		db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&user)
	}
}
