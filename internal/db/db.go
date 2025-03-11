package db

import (
	"log"
	"os"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB   *gorm.DB
	once sync.Once
)

func ConnectDB() error {
	var err error

	once.Do(func() {
		dbHost := os.Getenv("DB_HOST")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbPort := os.Getenv("DB_PORT")
		dbSSLMode := os.Getenv("DB_SSLMODE")
		dbTimezone := os.Getenv("DB_TIMEZONE")

		dsn := "host=" + dbHost +
			" user=" + dbUser +
			" password=" + dbPassword +
			" dbname=" + dbName +
			" port=" + dbPort +
			" sslmode=" + dbSSLMode +
			" TimeZone=" + dbTimezone

		var dbInstance *gorm.DB
		dbInstance, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Println(" Failed to connect to database:", err)
			return
		}

		DB = dbInstance
		log.Println(" Connected to database")
	})

	return err
}
