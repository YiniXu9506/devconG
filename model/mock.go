package model

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MockPhraseClick(n int, db *gorm.DB) {
	for i := 1; i <= n; i++ {
		phraseClick := PhraseClickModel{
			ID:        i,
			GroupID:   rand.Intn(5) + 1,
			OpenID:    fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
			PhraseID:  i,
			Clicks:    rand.Intn(100),
			ClickTime: time.Now(),
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
			Status:     1,
			CreateTime: time.Now(),
			UpdateTime:   time.Now(),
		}

		db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&phrase)
	}
}
