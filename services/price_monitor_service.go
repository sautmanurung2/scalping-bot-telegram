package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// PriceMonitorService memantau harga real-time gratis dan menyebarkan alert jika menyentuh SL/TP
type PriceMonitorService struct {
	db             *gorm.DB
	marketService  MarketServiceInterface
	telegramClient clients.TelegramClientInterface
	botToken       string
}

// NewPriceMonitorService menginisialisasi PriceMonitorService
func NewPriceMonitorService(
	db *gorm.DB,
	marketService MarketServiceInterface,
	telegramClient clients.TelegramClientInterface,
	botToken string,
) *PriceMonitorService {
	return &PriceMonitorService{
		db:             db,
		marketService:  marketService,
		telegramClient: telegramClient,
		botToken:       botToken,
	}
}

// StartMonitorDaemon menjalankan background ticker pemantau harga darurat
func (s *PriceMonitorService) StartMonitorDaemon(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	log.Println("🛡️ Real-time Price Alert Monitor Daemon dinyalakan...")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkPriceAlerts(ctx)
		}
	}
}

// checkPriceAlerts membandingkan running price dengan level Stop Loss / TP pada SQLite
func (s *PriceMonitorService) checkPriceAlerts(ctx context.Context) {
	if s.db == nil {
		return
	}

	var openJournals []models.TradingJournal
	err := s.db.WithContext(ctx).Where("status = ?", "OPEN").Find(&openJournals).Error
	if err != nil || len(openJournals) == 0 {
		return
	}

	var subs []models.BotSubscriber
	_ = s.db.WithContext(ctx).Where("is_active = ?", true).Find(&subs)
	if len(subs) == 0 {
		return
	}

	for _, j := range openJournals {
		var currentPrice float64

		if j.Market == models.MarketIndodax {
			ticker, err := s.marketService.FetchIndodaxTicker(ctx, j.Symbol)
			if err == nil && ticker != nil {
				fmt.Sscanf(ticker.Ticker.Last, "%f", &currentPrice)
			}
		} else {
			// Market IDX
			summary, err := s.marketService.FetchIDXSummary(ctx)
			if err == nil {
				for _, item := range summary {
					if item.Symbol == j.Symbol {
						currentPrice = item.LastPrice
						break
					}
				}
			}
		}

		if currentPrice <= 0 {
			continue
		}

		// Cek Pemicu Alert (Stop Loss atau TP)
		var alertType string
		var newStatus string

		if currentPrice <= j.StopPrice && j.StopPrice > 0 {
			alertType = "🛑 <b>PRICE ALERT: STOP LOSS TOUCHED!</b>"
			newStatus = "HIT_SL"
		} else if currentPrice >= j.TargetPrice && j.TargetPrice > 0 {
			alertType = "🎯 <b>PRICE ALERT: TAKE PROFIT TOUCHED!</b>"
			newStatus = "WIN_TP1"
		}

		if alertType != "" {
			// Update status di SQLite
			pnl := currentPrice - j.EntryPrice
			s.db.WithContext(ctx).Model(&j).Updates(map[string]interface{}{
				"status":      newStatus,
				"profit_loss": pnl,
			})

			alertMsg := fmt.Sprintf(
				"%s\n"+
					"----------------------------------------\n"+
					"<b>Aset:</b> %s (%s)\n"+
					"<b>Harga Entri:</b> Rp%.2f\n"+
					"<b>Harga Saat Ini:</b> Rp%.2f\n"+
					"<b>Estimasi PnL:</b> Rp%.2f\n\n"+
					"💡 <i>Segera amankan posisi / jalankan aturan manajemen risiko!</i>",
				alertType, j.Symbol, j.Market, j.EntryPrice, currentPrice, pnl,
			)

			// Kirim Notifikasi Darurat ke seluruh subscriber Telegram
			for _, sub := range subs {
				_ = s.telegramClient.SendMessage(ctx, s.botToken, sub.ChatID, alertMsg)
			}
		}
	}
}
