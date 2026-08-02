package models

import (
	"gorm.io/gorm"
)

// BotSubscriber merepresentasikan pengguna Telegram yang mengaktifkan Auto-Pilot Daily Scalping Bot
type BotSubscriber struct {
	gorm.Model
	ChatID    string `gorm:"type:varchar(50);uniqueIndex;not null" json:"chat_id"`
	FirstName string `gorm:"type:varchar(100)" json:"first_name"`
	IsActive  bool   `gorm:"default:true" json:"is_active"`
}
