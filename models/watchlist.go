package models

import (
	"gorm.io/gorm"
)

// UserWatchlist merepresentasikan entitas daftar pantau pribadi pengguna Telegram
type UserWatchlist struct {
	gorm.Model
	ChatID string     `gorm:"type:varchar(50);index;not null" json:"chat_id"`
	Symbol string     `gorm:"type:varchar(20);not null" json:"symbol"`
	Market MarketType `gorm:"type:varchar(20);not null" json:"market"`
}

// UserPriceAlert merepresentasikan entitas alarm harga kustom per pengguna
type UserPriceAlert struct {
	gorm.Model
	ChatID      string     `gorm:"type:varchar(50);index;not null" json:"chat_id"`
	Symbol      string     `gorm:"type:varchar(20);not null" json:"symbol"`
	Market      MarketType `gorm:"type:varchar(20);not null" json:"market"`
	TargetPrice float64    `gorm:"type:decimal(18,2);not null" json:"target_price"`
	Condition   string     `gorm:"type:varchar(10);not null" json:"condition"` // "ABOVE" atau "BELOW"
	IsTriggered bool       `gorm:"default:false" json:"is_triggered"`
}
