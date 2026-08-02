# 🚀 Trading Analysis Bot - AI Daily Scalping Telegram Bot

## 📋 Tentang Proyek
Trading Analysis Bot adalah aplikasi Telegram Bot cerdas berbasis Golang yang dirancang khusus untuk trader **Scalping Harian (Intraday Trading 5M/15M)** di pasar saham Indonesia (**IDX**) dan pasar kripto (**Indodax**).

Bot ini menggabungkan:
1. **Engine Analisis Teknikal Real-time**: Indikator RSI(14), MACD, EMA 9/21, dan Summary Rating dari TradingView.
2. **Senior Scalper AI Mentor**: Prompt engineering profesional yang memberikan **Kartu Saran Scalping Harian AI** lengkap dengan bias pasar, saran order limit, manajemen risiko tight stop loss (-0.8% s/d -1.2%), dan target TP1/TP2.
3. **Auto-Pilot Mode (`/start`)**: Memicu pemindaian harian otomatis instan untuk seluruh saham Indonesia (IDX) & Indodax secara dinamis serta daemon scheduler harian tanpa input manual berulang.
4. **Real-time Price Alert Monitor**: Memantau posisi harga running secara terus menerus dan mengirim notifikasi darurat saat tersentuh Stop Loss atau Take Profit.
5. **Multi-AI Circuit Breaker**: Failover otomatis jika AI Provider utama mengalami timeout/error.
6. **Jurnal & Performa Trading (`/journal` & `/stats`)**: Pencatatan riwayat sinyal di SQLite serta kalkulasi Win Rate (%), Total PnL, dan Profit Factor.

---

## 🏗️ Struktur Arsitektur Proyek

```text
.
├── clients/
│   ├── ai_client.go            # REST Client ke AI Provider dengan Multi-AI Circuit Breaker
│   ├── idx_client.go           # Scraper / Fetcher Data Saham IDX
│   ├── indodax_client.go       # Public REST Client Ticker Indodax
│   ├── telegram_client.go      # Telegram Bot API Client
│   └── tradingview_client.go   # Free TradingView Technical Scanner Client
├── config/
│   └── database.go             # Koneksi GORM SQLite & AutoMigrate
├── models/
│   ├── journal.go              # Model GORM TradingJournal
│   ├── market.go               # Struct Market Data & Tickers
│   ├── signal.go               # Struct ScanRequest & ScanResponse
│   └── subscriber.go           # Model GORM BotSubscriber (Auto-Pilot)
├── services/
│   ├── ai_advisor_service.go   # Service Prompt Engineering AI Scalper Mentor
│   ├── bot_dispatcher.go       # Telegram Long-Polling Engine & Handler Command
│   ├── journal_service.go      # Service Jurnal & Kalkulasi Win Rate / PnL
│   ├── market_service.go       # Service Fetcher Ticker IDX & Indodax
│   ├── notification_service.go # Service Formatter Notifikasi Telegram
│   ├── price_monitor_service.go# Service Real-time Price Alert Monitor Daemon
│   ├── signal_service.go       # Core Trading Signal Generator Engine
│   └── trader_engine.go        # Senior Trader Calculation Algorithm Engine
├── .env.example                # Template Variabel Lingkungan
├── docker-compose.yml          # Konfigurasi Docker Orchestration
├── Dockerfile                  # Multi-stage Docker Build File Golang
├── main.go                     # Entry Point Aplikasi & Dependency Injection
└── README.md                   # Dokumen Panduan & Penggunaan Proyek
```

---

## ⚙️ Persyaratan Sistem (Prerequisites)

- **Golang**: v1.23+ / v1.25+
- **Database**: SQLite3 (Otomatis dibuat oleh GORM sebagai `trading_bot.db`)
- **Telegram Bot Token**: Didapatkan via `@BotFather` di Telegram.
- **AI API Key**: OpenAI API Key atau Gemini API Key (atau provider AI kompatibel OpenAI).

---

## 🚀 Panduan Instalasi & Cara Menjalankan

### 1. Clone & Persiapan Repositori
```bash
git clone https://github.com/user/trading-analysis-bot.git
cd project-iseng
```

### 2. Konfigurasi File Environment `.env`
Salin file `.env.example` menjadi `.env`:
```bash
cp .env.example .env
```

Edit file `.env` dan masukkan kredensial Anda:
```env
TELEGRAM_BOT_TOKEN=8123456789:AAFxXXXXXX_YourTelegramBotToken
AI_PROVIDER_URL=https://api.openai.com/v1/chat/completions
AI_API_KEY=sk-proj-YourOpenAIKey
AI_MODEL=gpt-4o-mini

# Opsional: Secondary AI Provider untuk Circuit Breaker Failover
SECONDARY_AI_PROVIDER_URL=https://generativelanguage.googleapis.com/v1beta/openai/chat/completions
SECONDARY_AI_API_KEY=YourGeminiApiKey
SECONDARY_AI_MODEL=gemini-1.5-flash
```

### 3. Menjalankan Aplikasi

#### Cara 1: Menggunakan Go CLI Langsung
```bash
go run main.go
```

#### Cara 2: Menggunakan Docker Compose
```bash
docker-compose up -d --build
```

---

## 🤖 Perintah Telegram Bot (Commands Guide)

| Perintah | Deskripsi & Contoh Penggunaan |
| :--- | :--- |
| `/start` | **Mengaktifkan Auto-Pilot Daily Scalping Mode**. Memicu pemindaian dinamis instan untuk seluruh saham Indonesia (IDX) dan seluruh koin kripto Indodax ber-volume tinggi/potensial scalping harian, serta mendaftarkan scheduler otomatis. |
| `/stop` | **Menghentikan Auto-Pilot Daily Scalping Mode**. Menonaktifkan pendaftaran pengiriman notifikasi sinyal harian otomatis untuk pengguna. |
| `/scalp <market> <symbol>` | Analisis sinyal teknikal harian manual.<br>*Contoh:* `/scalp idx BBCA` atau `/scalp indodax btc_idr` |
| `/ai <market> <symbol>` | Meminta **Kartu Saran Scalping AI Mentor**.<br>*Contoh:* `/ai BBCA` atau `/ai indodax eth_idr` |
| `/ask <pertanyaan>` | Konsultasi strategi/psikologi trading dengan AI Mentor.<br>*Contoh:* `/ask bagaimana cara mengatasi FOMO saat scalping?` |
| `/news <symbol>` | Ringkasan berita finansial terkini & scoring sentimen AI (BULLISH/BEARISH/NEUTRAL).<br>*Contoh:* `/news BBCA` |
| `/bandar <symbol>` | Deteksi ketebalan Bid/Ask Orderbook & lonjakan volume harian vs 5-day average.<br>*Contoh:* `/bandar btc_idr` |
| `/export` | Mengunduh dokumen berkas rekap jurnal trading SQLite dalam bentuk file `.CSV`. |
| `/watch <add\|list\|del>` | Pengelolaan daftar pantau (*watchlist*) pribadi.<br>*Contoh:* `/watch add BBCA` atau `/watch list` |
| `/alert <symbol> <harga>` | Memasang alarm harga kustom personal.<br>*Contoh:* `/alert BBCA 10200` |
| `/journal` | Menampilkan 5 riwayat catatan jurnal trading terbaru dari SQLite. |
| `/stats` | Menampilkan statistik analisis performa: **Win Rate (%)**, Total PnL (Rp), dan Profit Factor. |
| `/idx` | Ringkasan harga & persentase saham teratas di Bursa Efek Indonesia. |
| `/indodax <pair>` | Cek harga & volume 24 jam pasar kripto Indodax.<br>*Contoh:* `/indodax btc_idr` |
| `/history` | Menampilkan 5 riwayat sinyal scalping harian terbaru. |
| `/help` | Menampilkan menu bantuan perintah Telegram. |
