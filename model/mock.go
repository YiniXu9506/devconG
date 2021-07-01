package model

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
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

		db.Create(&phraseClick)
	}
}

func MockPhrase(n int, db *gorm.DB) {
	for i := 1; i <= n; i++ {
		phrase := PhraseModel{
			PhraseID:       i,
			Text:           fmt.Sprintf("tidb%v", i),
			GroupID:        rand.Intn(5) + 1,
			OpenID:         fmt.Sprintf("%d", (rand.Intn(5)+1)*100),
			IsShow:         rand.Intn(2) == 1,
			IsDelete:       rand.Intn(2) == 1,
			CreateTime:     time.Now(),
			Clicks:         rand.Intn(10) + 30,
			HotGroupID:     rand.Intn(5) + 1,
			HotGroupClicks: rand.Intn(10) + 10,
		}

		db.Create(&phrase)
	}
}
