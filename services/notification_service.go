package services

import (
	"context"
	"fmt"
	"log"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// NotificationServiceInterface mendefinisikan kontrak pengelolaan notifikasi
type NotificationServiceInterface interface {
	SendSignalAlert(ctx context.Context, signal *models.ScanResponse) error
	SendTestAlert(ctx context.Context, botToken, chatID, message string) error
}

// NotificationService menangani pemformatan dan pengiriman notifikasi sinyal
type NotificationService struct {
	telegramClient clients.TelegramClientInterface
}

// NewNotificationService menginisialisasi NotificationService
func NewNotificationService(telegramClient clients.TelegramClientInterface) *NotificationService {
	return &NotificationService{
		telegramClient: telegramClient,
	}
}

// SendSignalAlert memformat sinyal rekomendasi trader dan mengiring notifikasi ke Telegram
func (s *NotificationService) SendSignalAlert(ctx context.Context, signal *models.ScanResponse) error {
	if signal == nil {
		return nil
	}

	icon := "📊"
	switch signal.Recommendation {
	case models.ActionStrongBuy:
		icon = "🔥"
	case models.ActionBuy:
		icon = "✅"
	case models.ActionSell:
		icon = "⚠️"
	case models.ActionAvoid:
		icon = "⛔"
	case models.ActionHold:
		icon = "⏳"
	}

	msg := fmt.Sprintf(
		"<b>%s TRADING SIGNAL ALERT: %s</b>\n"+
			"----------------------------------------\n"+
			"<b>Pasar:</b> %s | <b>Simbol:</b> %s\n"+
			"<b>Rekomendasi:</b> <u>%s</u>\n\n"+
			"🎯 <b>Entry Zone:</b> %.2f - %.2f\n"+
			"🛑 <b>Stop Loss:</b> %.2f\n"+
			"🎯 <b>Take Profit 1:</b> %.2f\n"+
			"🚀 <b>Take Profit 2:</b> %.2f\n"+
			"⚖️ <b>Risk-to-Reward:</b> %s\n\n"+
			"📈 <b>Ringkasan Indikator:</b>\n"+
			"• RSI (14): %.1f\n"+
			"• MACD: %s\n"+
			"• EMA Trend: %s\n"+
			"• TradingView: %s\n\n"+
			"📝 <b>Catatan Trader Senior:</b>\n<i>%s</i>",
		icon, signal.Symbol,
		signal.Market, signal.Symbol,
		signal.Recommendation,
		signal.EntryZone.Min, signal.EntryZone.Max,
		signal.StopLoss,
		signal.TakeProfit1,
		signal.TakeProfit2,
		signal.RiskRewardRatio,
		signal.TechnicalSummary.RSI14,
		signal.TechnicalSummary.MACDStatus,
		signal.TechnicalSummary.EMATrend,
		signal.TechnicalSummary.TradingViewRating,
		signal.TraderNotes,
	)

	// Panggil Telegram API
	err := s.telegramClient.SendMessage(ctx, "", "", msg)
	if err != nil {
		log.Printf("[NOTIFICATION WARNING] Gagal mengirim alert Telegram untuk %s: %v", signal.Symbol, err)
		return err
	}

	log.Printf("[NOTIFICATION SUCCESS] Alert Telegram sinyal %s (%s) berhasil terkirim.", signal.Symbol, signal.Recommendation)
	return nil
}

// SendTestAlert mengirimkan pesan ujicoba ke Telegram
func (s *NotificationService) SendTestAlert(ctx context.Context, botToken, chatID, message string) error {
	if message == "" {
		message = "🔔 <b>Test Notifikasi Financial Market Bot</b>\nKoneksi Telegram Bot API berhasil dihubungkan dengan sukses."
	}
	return s.telegramClient.SendMessage(ctx, botToken, chatID, message)
}
