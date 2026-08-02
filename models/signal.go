package models

import (
	"time"

	"gorm.io/gorm"
)

// SignalAction mendefinisikan tipe aksi rekomendasi trader
type SignalAction string

const (
	ActionStrongBuy SignalAction = "STRONG_BUY"
	ActionBuy       SignalAction = "BUY"
	ActionHold      SignalAction = "HOLD"
	ActionSell      SignalAction = "SELL"
	ActionAvoid     SignalAction = "AVOID"
)

// TradingSignal merepresentasikan entitas sinyal trading yang dihasilkan oleh Senior Trader Engine
type TradingSignal struct {
	gorm.Model
	Symbol           string       `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Market           MarketType   `gorm:"type:varchar(20);index;not null" json:"market"`
	Timeframe        string       `gorm:"type:varchar(10);not null" json:"timeframe"`
	Action           SignalAction `gorm:"type:varchar(20);not null" json:"action"`
	EntryMin         float64      `gorm:"type:decimal(18,2)" json:"entry_min"`
	EntryMax         float64      `gorm:"type:decimal(18,2)" json:"entry_max"`
	StopLoss         float64      `gorm:"type:decimal(18,2)" json:"stop_loss"`
	TakeProfit1      float64      `gorm:"type:decimal(18,2)" json:"take_profit_1"`
	TakeProfit2      float64      `gorm:"type:decimal(18,2)" json:"take_profit_2"`
	RiskRewardRatio  string       `gorm:"type:varchar(20)" json:"risk_reward_ratio"`
	RSI14            float64      `gorm:"type:decimal(8,2)" json:"rsi_14"`
	MACDStatus       string       `gorm:"type:varchar(50)" json:"macd_status"`
	EMATrend         string       `gorm:"type:varchar(50)" json:"ema_trend"`
	TradingViewScore string       `gorm:"type:varchar(20)" json:"tradingview_score"`
	TraderNotes      string       `gorm:"type:text" json:"trader_notes"`
	GeneratedAt      time.Time    `gorm:"autoCreateTime" json:"generated_at"`
}

// ScanRequest DTO merepresentasikan payload permintaan analisis sinyal
type ScanRequest struct {
	Market    MarketType `json:"market" binding:"required"`
	Symbol    string     `json:"symbol" binding:"required"`
	Timeframe string     `json:"timeframe"` // Contoh: "1D", "4H", "1H"
}

// ScanResponse DTO merepresentasikan struktur keluaran hasil analisis sinyal trader
type ScanResponse struct {
	Symbol           string       `json:"symbol"`
	Market           MarketType   `json:"market"`
	Recommendation   SignalAction `json:"recommendation"`
	EntryZone        EntryZone    `json:"entry_zone"`
	StopLoss         float64      `json:"stop_loss"`
	TakeProfit1      float64      `json:"take_profit_1"`
	TakeProfit2      float64      `json:"take_profit_2"`
	RiskRewardRatio  string       `json:"risk_reward_ratio"`
	TechnicalSummary TechSummary  `json:"technical_summary"`
	TraderNotes      string       `json:"trader_notes"`
}

// EntryZone DTO untuk rentang harga masuk (buy zone)
type EntryZone struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// TechSummary DTO ringkasan indikator teknikal
type TechSummary struct {
	RSI14             float64 `json:"rsi_14"`
	MACDStatus        string  `json:"macd_status"`
	EMATrend          string  `json:"ema_trend"`
	TradingViewRating string  `json:"tradingview_rating"`
}
