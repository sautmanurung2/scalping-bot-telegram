package models

import (
	"time"

	"gorm.io/gorm"
)

// MarketType mendefinisikan tipe pasar (IDX atau INDODAX)
type MarketType string

const (
	MarketIDX     MarketType = "IDX"
	MarketIndodax MarketType = "INDODAX"
)

// MarketData merepresentasikan data cache harga pasar yang disimpan di SQLite via GORM
type MarketData struct {
	gorm.Model
	Symbol           string     `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Market           MarketType `gorm:"type:varchar(20);index;not null" json:"market"`
	LastPrice        float64    `gorm:"type:decimal(18,2)" json:"last_price"`
	High24h          float64    `gorm:"type:decimal(18,2)" json:"high_24h"`
	Low24h           float64    `gorm:"type:decimal(18,2)" json:"low_24h"`
	Volume           float64    `gorm:"type:decimal(18,2)" json:"volume"`
	PriceChange      float64    `gorm:"type:decimal(18,2)" json:"price_change"`
	PriceChangeRatio float64    `gorm:"type:decimal(8,4)" json:"price_change_ratio"`
	FetchedAt        time.Time  `gorm:"autoCreateTime" json:"fetched_at"`
}

// IDXTickerResponse DTO merepresentasikan respons JSON dari NeaByteLab/IDX-API
type IDXTickerResponse struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	LastPrice  float64 `json:"last_price"`
	Change     float64 `json:"change"`
	Percentage float64 `json:"percentage"`
	Volume     float64 `json:"volume"`
	Value      float64 `json:"value"`
}

// IndodaxTickerItem DTO merepresentasikan item ticker tunggal dari Indodax API
type IndodaxTickerItem struct {
	High       string `json:"high"`
	Low        string `json:"low"`
	VolIDR     string `json:"vol_idr"`
	Last       string `json:"last"`
	Buy        string `json:"buy"`
	Sell       string `json:"sell"`
	ServerTime int64  `json:"server_time"`
}

// IndodaxTickerResponse DTO merepresentasikan respons API Ticker Indodax
type IndodaxTickerResponse struct {
	Ticker IndodaxTickerItem `json:"ticker"`
}

// OrderbookEntry DTO merepresentasikan baris kedalaman pasar (Bid / Ask)
type OrderbookEntry struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// DepthData DTO merepresentasikan struktur Orderbook Depth dari Indodax / IDX
type DepthData struct {
	Bids []OrderbookEntry `json:"bids"`
	Asks []OrderbookEntry `json:"asks"`
}
