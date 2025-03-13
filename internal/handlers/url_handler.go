package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"net/http"
	"os"
	"strings"
	"url-shortener/internal/db"
	"url-shortener/internal/models"

	"context"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var (
	host        = os.Getenv("REDIS_HOST")
	port        = os.Getenv("REDIS_PORT")
	redisClient = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "",
		DB:       0,
	})
)

var ctx = context.Background()

type ShortenRequest struct {
	OrignalUrl string `json:"orignal_url" binding:"required"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

func GenerateShortcode() string {
	b := make([]byte, 4)
	fmt.Println(b)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")[:6]
}

func ShortenUrlHandler(c *gin.Context) {
	var req ShortenRequest

	ENDPOINT := os.Getenv("ENDPOINT")

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	cacheKey := "short_url:" + req.OrignalUrl
	cachedShortURL, err := redisClient.Get(ctx, cacheKey).Result()

	if err == nil {
		fmt.Println("displaying what has cached")
		c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + cachedShortURL})
		return
	}

	var existingURL models.URL
	db.DB.Where("orignal_url = ?", req.OrignalUrl).First(&existingURL)

	if existingURL.ID != 0 {
		redisClient.Set(ctx, cacheKey, existingURL.ShortUrl, 24*time.Hour)
		redisClient.Set(ctx, "short_code:"+existingURL.ShortUrl, req.OrignalUrl, 24*time.Hour)
		c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + existingURL.ShortUrl})
		return
	}

	var shortCode string
	for {
		shortCode = GenerateShortcode()
		if shortCode == "" {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save URL"})
		return
	}

	redisClient.Set(ctx, cacheKey, shortCode, 24*time.Hour)
	redisClient.Set(ctx, "short_code:"+shortCode, req.OrignalUrl, 24*time.Hour)

	c.JSON(http.StatusOK, ShortenResponse{ShortURL: ENDPOINT + shortCode})
}

func RedirectHandler(c *gin.Context) {
	shortCode := c.Param("shortCode")
	fmt.Println("ShortCode received:", shortCode)

	cacheKey := "short_code:" + shortCode
	orignalUrl, err := redisClient.Get(ctx, cacheKey).Result()
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

	redisClient.Set(ctx, cacheKey, shortCode, 24*time.Hour)
	c.Redirect(http.StatusFound, existingURL.OrignalUrl)

}
