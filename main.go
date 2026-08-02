package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/config"
	"trading-analysis-bot/services"
)

func main() {
	log.Println("🚀 Memulai Financial Market Analysis - AI Daily Scalping Telegram Bot...")

	// Context global dengan penanganan Signal Interruption (Ctrl+C / SIGTERM)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Inisialisasi Database SQLite via GORM
	db := config.InitDatabase()

	// 2. Inisialisasi API Clients Eksternal (IDX, Indodax, TradingView, Telegram, AI Provider)
	idxClient := clients.NewIDXClient()
	indodaxClient := clients.NewIndodaxClient()
	tvClient := clients.NewTradingViewClient()
	telegramClient := clients.NewTelegramClient()
	aiClient := clients.NewAIClient()

	// 3. Inisialisasi Core Services & Senior Trader Scalper Engine
	traderEngine := services.NewTraderEngine()
	aiAdvisorService := services.NewAIAdvisorService(aiClient)
	marketService := services.NewMarketService(db, idxClient, indodaxClient)
	signalService := services.NewSignalService(db, idxClient, indodaxClient, tvClient, traderEngine, nil)
	journalService := services.NewJournalService(db)
	newsService := services.NewNewsService(aiClient)
	bandarService := services.NewBandarmologiService(indodaxClient, idxClient)

	// 4. Inisialisasi Telegram Bot Polling Engine & Dispatcher
	botDispatcher := services.NewBotDispatcher(telegramClient, signalService, marketService, aiAdvisorService, journalService, newsService, bandarService)

	log.Println("✅ Inisialisasi komponen selesai. Menjalankan Telegram Bot Daily Scalping Daemon...")

	// Jalankan Telegram Bot Polling Engine (Mendengarkan pesan Telegram sampai Ctrl+C / Interrupt)
	botDispatcher.StartPolling(ctx)

	log.Println("👋 Telegram Bot berhasil dihentikan secara aman.")
}
