package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logFile, err := os.OpenFile("/tmp/url-shortener.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		logger.Fatal("Failed to open log file", err)
	}

	logger.SetOutput(logFile)

	err = godotenv.Load()

	if err != nil {
		logger.Errorf("Error loading .env file: %v", err)
		log.Println("Warning: .env file not found, using default environment variables")

	}

	if err := db.ConnectDB(); err != nil {
		log.Fatal("Database connection failed")
	}

	if db.DB == nil {
		logger.Fatal("Database connection is nil")
	}

	if err := db.DB.AutoMigrate(&models.URL{}); err != nil {
		log.Fatal("Migration failed:", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: "",
		DB:       0,
	})
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

	shortenHandler := &handlers.RedisHandle{RedisClient: redisClient}
	redirectHandler := &handlers.RedisHandle{RedisClient: redisClient}

	r.Use(handlers.RateLimiterMiddleware(redisClient, 5, time.Minute))

	r.POST("/url/shorten", shortenHandler.ShortenUrlHandler)
	r.GET("/:shortCode", redirectHandler.RedirectHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Listening on port:", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error starting server:", err)
	}

}
