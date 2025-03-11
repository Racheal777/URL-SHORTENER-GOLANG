package models

import "gorm.io/gorm"

type URL struct {
	gorm.Model
	OrignalUrl string `gorm:"uniqueIndex;not null"`
	ShortUrl   string `gorm:"uniqueIndex;not null"`
}
