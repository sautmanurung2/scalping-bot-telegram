package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"trading-analysis-bot/models"
)

// DB adalah instance koneksi database global GORM
var DB *gorm.DB

// InitDatabase menginisialisasi koneksi SQLite dan menjalankan migrasi skema tabel
func InitDatabase() *gorm.DB {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "trading_bot.db"
	}

	// Pastikan direktori induk tempat file database berada tersedia
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Gagal membuat direktori database %s: %v", dir, err)
		}
	}

	// Konfigurasi koneksi GORM dengan SQLite
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Gagal terhubung ke database SQLite: %v", err)
		os.Exit(1)
	}

	log.Println("Koneksi database SQLite berhasil diinisialisasi.")

	// Otomatis jalankan migrasi struktur tabel (AutoMigrate)
	err = db.AutoMigrate(
		&models.MarketData{},
		&models.TradingSignal{},
		&models.TradingJournal{},
		&models.BotSubscriber{},
		&models.UserWatchlist{},
		&models.UserPriceAlert{},
	)
	if err != nil {
		log.Fatalf("Gagal melakukan otomatisasi migrasi database: %v", err)
	}

	log.Println("Migrasi tabel (MarketData, TradingSignal, TradingJournal, BotSubscriber, UserWatchlist, UserPriceAlert) berhasil dikonfirmasi.")

	DB = db
	return db
}
