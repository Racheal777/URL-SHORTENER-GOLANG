package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"os"
	"url-shortener/internal/handlers"

	"log"
	_ "os"
	"strconv"
	"time"
	"url-shortener/internal/db"
	_ "url-shortener/internal/handlers"
	"url-shortener/internal/metrics"
	"url-shortener/internal/models"
)

func main() {

	err := godotenv.Load()
	log.Println(err)
	if err != nil {
		log.Println("Warning: .env file not found, using default environment variables")

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

	r.Use(func(c *gin.Context) {
		start := time.Now()
		statusCode := c.Writer.Status()
		duration := time.Since(start).Seconds()
		method := c.Request.Method
		endpoint := c.FullPath()
		status := strconv.Itoa(statusCode)

		// Record the request
		metrics.RequestTotal.WithLabelValues(method, endpoint, status).Inc()
		metrics.RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)

		// Record request size (if Content-Length is available)
		if requestSize := c.Request.ContentLength; requestSize > 0 {
			metrics.RequestSize.WithLabelValues(method, endpoint, status).Observe(float64(requestSize))
		}

		// Record response size
		if responseSize := c.Writer.Size(); responseSize > 0 {
			metrics.ResponseSize.WithLabelValues(method, endpoint, status).Observe(float64(responseSize))
		}

	})
	r.POST("/url/shorten", handlers.ShortenUrlHandler)
	r.GET("/:shortCode", handlers.RedirectHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Listening on port:", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}

}
