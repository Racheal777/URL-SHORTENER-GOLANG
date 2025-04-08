package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"

	"time"

	"net/http"
	"os"
	"strings"
	"url-shortener/internal/db"
	"url-shortener/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

//var (
//	host        = os.Getenv("REDIS_HOST")
//	port        = os.Getenv("REDIS_PORT")
//	redisClient = redis.NewClient(&redis.Options{
//		Addr:     host + ":" + port,
//		Password: "",
//		DB:       0,
//	})
//)

// var ctx = context.Background()
var logger = logrus.New()

func RateLimiterMiddleware(client *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := c.Request.Context()
		log.Print("fetching IP", ip)
		key := fmt.Sprintf("ratelimit:%s", ip)

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		if count == 1 {
			client.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			c.Abort()
			return
		}

		c.Next()

	}
}

type ShortenRequest struct {
	OrignalUrl string `json:"orignal_url" binding:"required"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

type RedisHandle struct {
	RedisClient *redis.Client
}

func GenerateShortcode() string {
	b := make([]byte, 4)
	fmt.Println(b)
	_, err := rand.Read(b)
	if err != nil {
		logger.WithError(err).Error("Error generating shortcode")
		return ""
	}
	shortCode := strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")[:6]
	logger.WithField("shortcode", shortCode).Info("Generated shortcode successfully")
	return shortCode
}

func (h *RedisHandle) ShortenUrlHandler(c *gin.Context) {
	var req ShortenRequest

	ENDPOINT := os.Getenv("ENDPOINT")
	ctx := c.Request.Context()

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithError(err).Warn("Invalid request body received")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	logger.WithField("original_url", req.OrignalUrl).Info("Received request to shorten URL")
	cacheKey := "short_url:" + req.OrignalUrl
	cachedShortURL, err := h.RedisClient.Get(ctx, cacheKey).Result()

	if err == nil {
		logger.WithField("short_url", cachedShortURL).Info("Serving from cache")
		c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + cachedShortURL})
		return
	}

	var existingURL models.URL
	db.DB.Where("orignal_url = ?", req.OrignalUrl).First(&existingURL)

	if existingURL.ID != 0 {
		logger.WithFields(logrus.Fields{
			"short_url":    existingURL.ShortUrl,
			"original_url": req.OrignalUrl,
		}).Info("URL already exists, returning cached value")

		h.RedisClient.Set(ctx, cacheKey, existingURL.ShortUrl, 24*time.Hour)
		h.RedisClient.Set(ctx, "short_code:"+existingURL.ShortUrl, req.OrignalUrl, 24*time.Hour)
		c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + existingURL.ShortUrl})
		return
	}

	var shortCode string
	for {
		shortCode = GenerateShortcode()
		if shortCode == "" {
			logger.Error("Failed to generate a short code")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate short code"})
			return
		}

		var existingShort models.URL
		db.DB.Where("short_url = ?", shortCode).First(&existingShort)

		if existingShort.ID == 0 {
			break
		}
	}

	url := models.URL{OrignalUrl: req.OrignalUrl, ShortUrl: shortCode}
	result := db.DB.Create(&url)

	if result.Error != nil {
		logger.WithError(result.Error).Error("Failed to save URL to database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	logger.WithFields(logrus.Fields{
		"original_url": req.OrignalUrl,
		"short_url":    shortCode,
	}).Info("Successfully shortened URL")

	h.RedisClient.Set(ctx, cacheKey, shortCode, 24*time.Hour)
	h.RedisClient.Set(ctx, "short_code:"+shortCode, req.OrignalUrl, 24*time.Hour)

	c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + shortCode})
	return
}

func (h *RedisHandle) RedirectHandler(c *gin.Context) {
	shortCode := c.Param("shortCode")
	ctx := c.Request.Context()
	fmt.Println("ShortCode received:", shortCode)

	cacheKey := "short_code:" + shortCode
	orignalUrl, err := h.RedisClient.Get(ctx, cacheKey).Result()
	fmt.Println("cache received:", orignalUrl)

	if err == nil {
		c.Redirect(http.StatusFound, orignalUrl)
		return
	}

	var existingURL models.URL
	result := db.DB.Where("short_url = ?", shortCode).First(&existingURL)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	h.RedisClient.Set(ctx, cacheKey, shortCode, 24*time.Hour)
	c.Redirect(http.StatusFound, existingURL.OrignalUrl)
	return

}
