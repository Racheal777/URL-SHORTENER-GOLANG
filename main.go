package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
	"url-shortener/internal/db"
	"url-shortener/internal/handlers"
	"url-shortener/internal/models"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := db.ConnectDB(); err != nil {
		log.Fatal("Database connection failed")
	}

	if db.DB == nil {
		log.Fatal("Database connection is nil")
	}

	if err := db.DB.AutoMigrate(&models.URL{}); err != nil {
		log.Fatal("Migration failed:", err)
	}

	r := gin.Default()
	r.POST("/url/shorten", handlers.ShortenUrlHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Listening on port:", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}

}
