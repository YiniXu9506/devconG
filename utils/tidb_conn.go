package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/YiniXu9506/devconG/model"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gorm.io/gorm/logger"
)

func TiDBConnect(hostName string, port int) *gorm.DB {
	dsn := fmt.Sprintf("root@tcp(%v:%v)/test?charset=utf8mb4&parseTime=True&loc=Local", hostName, port)
	fmt.Printf("dsn %v\n", dsn)
	start := time.Now()
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
		panic(fmt.Sprintf("failed to connect database %v", err))
	}

	db.AutoMigrate(&model.PhraseClickModel{}, &model.PhraseModel{}, &model.UserModel{})

	zap.L().Sugar().Infof("migrate db cost: %v\n", time.Since(start))
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to connect database %v", err))
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(1000)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)
	// start = time.Now()
	model.MockPhraseClick(50, db)

	model.MockPhrase(50, db)

	model.MockUser(50, db)
	// zap.L().Sugar().Infof("mock cost: %v\n", time.Since(start))
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
