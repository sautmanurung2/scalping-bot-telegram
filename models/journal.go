package models

import (
	"time"

	"gorm.io/gorm"
)

// TradingJournal merepresentasikan entitas jurnal performa trading yang dicatat oleh bot
type TradingJournal struct {
	gorm.Model
	SignalID    uint       `gorm:"index" json:"signal_id"`
	Symbol      string     `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Market      MarketType `gorm:"type:varchar(20);not null" json:"market"`
	EntryPrice  float64    `gorm:"type:decimal(18,2)" json:"entry_price"`
	TargetPrice float64    `gorm:"type:decimal(18,2)" json:"target_price"`
	StopPrice   float64    `gorm:"type:decimal(18,2)" json:"stop_price"`
	Status      string     `gorm:"type:varchar(20);default:'OPEN'" json:"status"` // "OPEN", "WIN_TP1", "WIN_TP2", "HIT_SL", "CLOSED"
	ProfitLoss  float64    `gorm:"type:decimal(18,2)" json:"profit_loss"`
	Notes       string     `gorm:"type:text" json:"notes"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}
