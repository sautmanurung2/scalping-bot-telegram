package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/config"
	"trading-analysis-bot/models"
)

// BotDispatcherInterface mendefinisikan kontrak pengelola perintah Telegram Bot
type BotDispatcherInterface interface {
	StartPolling(ctx context.Context)
}

// BotDispatcher mengelola Long-Polling dan mengeksekusi perintah dari user di Telegram
type BotDispatcher struct {
	botToken         string
	telegramClient   clients.TelegramClientInterface
	signalService    SignalServiceInterface
	marketService    MarketServiceInterface
	aiAdvisorService AIAdvisorServiceInterface
	journalService   JournalServiceInterface
	newsService      NewsServiceInterface
	bandarService    BandarmologiServiceInterface
}

// NewBotDispatcher menginisialisasi BotDispatcher
func NewBotDispatcher(
	telegramClient clients.TelegramClientInterface,
	signalService SignalServiceInterface,
	marketService MarketServiceInterface,
	aiAdvisorService AIAdvisorServiceInterface,
	journalService JournalServiceInterface,
	newsService NewsServiceInterface,
	bandarService BandarmologiServiceInterface,
) *BotDispatcher {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	return &BotDispatcher{
		botToken:         token,
		telegramClient:   telegramClient,
		signalService:    signalService,
		marketService:    marketService,
		aiAdvisorService: aiAdvisorService,
		journalService:   journalService,
		newsService:      newsService,
		bandarService:    bandarService,
	}
}

// StartPolling menjalankan loop Long-Polling secara terus menerus (daemon process)
func (b *BotDispatcher) StartPolling(ctx context.Context) {
	if b.botToken == "" {
		b.botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	log.Println("🤖 Telegram Bot Daily Scalping Engine dinyalakan...")
	if b.botToken == "" {
		log.Println("⚠️ WARNING: TELEGRAM_BOT_TOKEN belum diatur! Bot akan menunggu kredensial diatur.")
	}

	// Jalankan background scheduler harian otomatis
	go b.startDailyScheduler(ctx)

	// Jalankan real-time price alert monitor daemon
	priceMonitor := NewPriceMonitorService(config.DB, b.marketService, b.telegramClient, b.botToken)
	go priceMonitor.StartMonitorDaemon(ctx)

	var offset int64 = 0

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Menghentikan Telegram Bot Polling Engine...")
			return
		default:
			if b.botToken == "" {
				b.botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
				if b.botToken == "" {
					time.Sleep(5 * time.Second)
					continue
				}
			}

			updates, err := b.telegramClient.GetUpdates(ctx, b.botToken, offset, 20)
			if err != nil {
				log.Printf("[BOT ERROR] Gagal mengambil updates: %v. Mencoba lagi dalam 5 detik...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updates {
				if update.UpdateID >= offset {
					offset = update.UpdateID + 1
				}

				if update.Message != nil && update.Message.Text != "" {
					go b.processMessageSafe(ctx, update.Message)
				}
			}
		}
	}
}

// processMessageSafe mengeksekusi penanganan pesan dengan proteksi panic recovery
func (b *BotDispatcher) processMessageSafe(ctx context.Context, msg *clients.TelegramMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] Error pada penanganan pesan Telegram: %v", r)
		}
	}()

	chatIDStr := strconv.FormatInt(msg.Chat.ID, 10)
	rawText := strings.TrimSpace(msg.Text)
	parts := strings.Fields(rawText)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	if strings.Contains(command, "@") {
		command = strings.Split(command, "@")[0]
	}

	log.Printf("[BOT COMMAND] Menerima perintah '%s' dari User ID: %d (%s)", rawText, msg.From.ID, msg.From.FirstName)

	switch command {
	case "/start":
		b.handleStartCommand(ctx, chatIDStr, msg.From.FirstName)

	case "/help":
		b.handleHelpCommand(ctx, chatIDStr, msg.From.FirstName)

	case "/scalp", "/scan":
		b.handleScalpCommand(ctx, chatIDStr, parts[1:])

	case "/ai", "/advise":
		b.handleAIAdvisorCommand(ctx, chatIDStr, parts[1:])

	case "/ask":
		b.handleAskCommand(ctx, chatIDStr, strings.Join(parts[1:], " "))

	case "/idx":
		b.handleIDXCommand(ctx, chatIDStr)

	case "/indodax":
		b.handleIndodaxCommand(ctx, chatIDStr, parts[1:])

	case "/history":
		b.handleHistoryCommand(ctx, chatIDStr)

	case "/journal":
		b.handleJournalCommand(ctx, chatIDStr)

	case "/stats":
		b.handleStatsCommand(ctx, chatIDStr)

	case "/news":
		b.handleNewsCommand(ctx, chatIDStr, parts[1:])

	case "/bandar":
		b.handleBandarCommand(ctx, chatIDStr, parts[1:])

	case "/export":
		b.handleExportCommand(ctx, chatIDStr)

	case "/watch":
		b.handleWatchCommand(ctx, chatIDStr, parts[1:])

	case "/alert":
		b.handleAlertCommand(ctx, chatIDStr, parts[1:])

	default:
		if strings.HasPrefix(command, "/") {
			reply := fmt.Sprintf("⚠️ Perintah <code>%s</code> tidak dikenali. Ketik /help untuk melihat daftar perintah yang tersedia.", command)
			_ = b.telegramClient.SendMessage(ctx, b.botToken, chatIDStr, reply)
		}
	}
}

// handleHelpCommand mengirim pesan panduan pengguna
func (b *BotDispatcher) handleHelpCommand(ctx context.Context, chatID, firstName string) {
	helpText := fmt.Sprintf(
		"👋 <b>Selamat Datang di AI Daily Scalping Trading Bot, %s!</b>\n\n"+
			"Saya adalah Bot Scalping Harian (Intraday) berbasis AI & Senior Trader Engine (15+ Tahun Pengalaman).\n\n"+
			"📌 <b>Daftar Perintah Telegram Bot:</b>\n"+
			"• <code>/scalp &lt;MARKET&gt; &lt;SYMBOL&gt;</code> - Analisis cepat sinyal scalping harian (5M/15M).\n"+
			"  <i>Contoh:</i> <code>/scalp idx BBCA</code> atau <code>/scalp indodax btc_idr</code>\n\n"+
			"• <code>/ai &lt;MARKET&gt; &lt;SYMBOL&gt;</code> - Kartu Saran Scalping Harian Terintegrasi AI Provider.\n"+
			"  <i>Contoh:</i> <code>/ai BBCA</code> atau <code>/advise indodax eth_idr</code>\n\n"+
			"• <code>/ask &lt;PERTANYAAN&gt;</code> - Konsultasi strategi scalping & psikologi pasar dengan AI Mentor.\n"+
			"  <i>Contoh:</i> <code>/ask bagaimana menentukan batas cutloss scalping?</code>\n\n"+
			"• <code>/idx</code> - Ringkasan harga saham utama di Bursa Efek Indonesia (IDX).\n\n"+
			"• <code>/indodax &lt;pair&gt;</code> - Cek harga & volume 24 jam pasar kripto Indodax.\n\n"+
			"• <code>/history</code> - Lihat 5 riwayat sinyal scalping harian terbaru dari database.\n\n"+
			"• <code>/help</code> - Menampilkan menu bantuan ini.",
		firstName,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, helpText)
}

// handleStartCommand mendaftarkan subscriber dan memicu pemindaian scalping harian otomatis instan
func (b *BotDispatcher) handleStartCommand(ctx context.Context, chatID, firstName string) {
	b.registerSubscriber(chatID, firstName)

	startMsg := fmt.Sprintf(
		"🚀 <b>AUTO-PILOT DAILY SCALPING BOT AKTIF!</b>\n\n"+
			"Halo <b>%s</b>! Mode <b>Scalping Harian Otomatis</b> berbasis AI Provider & Senior Trader Engine kini telah <b>AKTIF</b>.\n\n"+
			"⚡ Bot akan langsung memproses pemindaian harian dan mengirimkan analisis sinyal scalping & kartu saran AI secara otomatis ke obrolan ini.",
		firstName,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, startMsg)

	// Jalankan pemindaian instan untuk user ini secara async
	go b.runAutoDailyScalpBatch(ctx, chatID)
}

// registerSubscriber menyimpan atau memperbarui status subscriber Telegram di database SQLite
func (b *BotDispatcher) registerSubscriber(chatID, firstName string) {
	if config.DB != nil {
		var sub models.BotSubscriber
		err := config.DB.Where("chat_id = ?", chatID).First(&sub).Error
		if err != nil {
			newSub := models.BotSubscriber{
				ChatID:    chatID,
				FirstName: firstName,
				IsActive:  true,
			}
			config.DB.Create(&newSub)
		} else {
			if !sub.IsActive {
				config.DB.Model(&sub).Update("is_active", true)
			}
		}
	}
}

// runAutoDailyScalpBatch memproses pemindaian sinyal & saran AI untuk seluruh saham Indonesia (IDX) & Indodax secara dinamis
func (b *BotDispatcher) runAutoDailyScalpBatch(ctx context.Context, chatID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] Error pada auto daily scalp batch: %v", r)
		}
	}()

	var assets []struct {
		Market models.MarketType
		Symbol string
	}

	// 1. Ambil seluruh daftar saham Indonesia (IDX) secara dinamis dari Market Service
	idxSummary, err := b.marketService.FetchIDXSummary(ctx)
	if err == nil && len(idxSummary) > 0 {
		for _, stock := range idxSummary {
			if stock.Symbol != "" {
				assets = append(assets, struct {
					Market models.MarketType
					Symbol string
				}{
					Market: models.MarketIDX,
					Symbol: stock.Symbol,
				})
			}
		}
	} else {
		// Fallback jika API utama terkendala
		fallbackSymbols := []string{
			"ADRO", "ANTM", "BIRD", "PADI", "PRDA", "PRDL", "PTBA", "PWON",
			"BBCA", "BBRI", "BMRI", "BBNI", "TLKM", "ASII", "AMMN", "GOTO",
			"UNVR", "ICBP", "INDF", "BRPT", "TPIA", "PGAS", "MDKA", "INKP",
			"MEDC", "CPIN", "KLBF", "UNTR",
		}
		for _, sym := range fallbackSymbols {
			assets = append(assets, struct {
				Market models.MarketType
				Symbol string
			}{Market: models.MarketIDX, Symbol: sym})
		}
	}

	// 2. Ambil seluruh koin kripto Indodax yang memiliki volume tebal & potensial scalping secara dinamis
	cryptoPairs, err := b.marketService.FetchIndodaxTopPairs(ctx)
	if err == nil && len(cryptoPairs) > 0 {
		for _, pair := range cryptoPairs {
			assets = append(assets, struct {
				Market models.MarketType
				Symbol string
			}{
				Market: models.MarketIndodax,
				Symbol: pair,
			})
		}
	} else {
		assets = append(assets,
			struct {
				Market models.MarketType
				Symbol string
			}{Market: models.MarketIndodax, Symbol: "btc_idr"},
			struct {
				Market models.MarketType
				Symbol string
			}{Market: models.MarketIndodax, Symbol: "eth_idr"},
		)
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID,
		fmt.Sprintf("📊 <b>MEMULAI PEMINDAIAN DINAMIS %d ASET (SELURUH SAHAM IDX & KOIN KRIPTO INDODAX)...</b>\n----------------------------------------\n<i>Sedang memproses indikator teknikal 15M/5M dan rekomendasi AI Mentor.</i>", len(assets)),
	)

	for _, item := range assets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		req := models.ScanRequest{
			Market:    item.Market,
			Symbol:    item.Symbol,
			Timeframe: "15M",
		}

		scanRes, err := b.signalService.GenerateSignal(ctx, req)
		if err != nil {
			log.Printf("[AUTO SCALP ERROR] Gagal generate signal untuk %s: %v", item.Symbol, err)
			continue
		}

		if b.aiAdvisorService != nil {
			adviceMsg, err := b.aiAdvisorService.GenerateScalpingAdvice(ctx, scanRes)
			if err == nil && adviceMsg != "" {
				_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, adviceMsg)
			}
		}

		if b.journalService != nil {
			_, _ = b.journalService.CreateJournalFromSignal(ctx, scanRes)
		}

		time.Sleep(500 * time.Millisecond)
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID,
		"✅ <b>PEMINDAIAN SCALPING HARIAN HARI INI SELESAI!</b>\n"+
			"----------------------------------------\n"+
			"🔔 <i>Bot akan otomatis memperbarui dan mengirimkan sinyal & saran AI scalping harian setiap hari bursa tanpa perlu input ulang. Selamat trading & amankan profit disiplin!</i>",
	)
}

// startDailyScheduler memantau jam bursa dan mengirimkan sinyal harian otomatis ke seluruh subscriber
func (b *BotDispatcher) startDailyScheduler(ctx context.Context) {
	ticker := time.NewTicker(4 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if config.DB == nil {
				continue
			}

			var subs []models.BotSubscriber
			err := config.DB.Where("is_active = ?", true).Find(&subs).Error
			if err != nil || len(subs) == 0 {
				continue
			}

			log.Printf("⏰ [SCHEDULER] Menjalankan pemindaian scalping harian otomatis untuk %d subscriber...", len(subs))

			for _, sub := range subs {
				go b.runAutoDailyScalpBatch(ctx, sub.ChatID)
			}
		}
	}
}

// handleScalpCommand menangani pemindaian sinyal scalping intraday harian
func (b *BotDispatcher) handleScalpCommand(ctx context.Context, chatID string, args []string) {
	if len(args) == 0 {
		reply := "⚠️ <b>Format Perintah Salah!</b>\n" +
			"Gunakan format: <code>/scalp &lt;MARKET&gt; &lt;SYMBOL&gt;</code>\n\n" +
			"<b>Contoh Saham IDX:</b>\n<code>/scalp idx BBCA</code>\n" +
			"<b>Contoh Kripto Indodax:</b>\n<code>/scalp indodax btc_idr</code>"
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, reply)
		return
	}

	market := models.MarketIDX
	symbol := args[0]

	if len(args) >= 2 {
		marketStr := strings.ToUpper(args[0])
		if marketStr == "INDODAX" || marketStr == "CRYPTO" {
			market = models.MarketIndodax
		}
		symbol = args[1]
	} else {
		if strings.Contains(strings.ToLower(symbol), "_idr") || strings.EqualFold(symbol, "btc") || strings.EqualFold(symbol, "eth") {
			market = models.MarketIndodax
		}
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("⚡ <b>Memproses Analisis Scalping Intraday (15M) untuk %s (%s)... Mohon tunggu.</b>", strings.ToUpper(symbol), market))

	req := models.ScanRequest{
		Market:    market,
		Symbol:    symbol,
		Timeframe: "15M",
	}

	res, err := b.signalService.GenerateSignal(ctx, req)
	if err != nil {
		reply := fmt.Sprintf("❌ <b>Gagal Melakukan Analisis Scalping:</b> %v", err)
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, reply)
		return
	}

	icon := "📊"
	switch res.Recommendation {
	case models.ActionStrongBuy:
		icon = "⚡"
	case models.ActionBuy:
		icon = "✅"
	case models.ActionSell:
		icon = "⚠️"
	case models.ActionAvoid:
		icon = "⛔"
	case models.ActionHold:
		icon = "⏳"
	}

	cardMsg := fmt.Sprintf(
		"<b>%s DAILY SCALPING SIGNAL: %s</b>\n"+
			"----------------------------------------\n"+
			"<b>Pasar:</b> %s | <b>Timeframe:</b> 15M / 5M\n"+
			"<b>Rekomendasi Scalper:</b> <u>%s</u>\n\n"+
			"🎯 <b>Entry Zone Scalp:</b> Rp%.2f - Rp%.2f\n"+
			"🛑 <b>Stop Loss Strict:</b> Rp%.2f (-1.0%%)\n"+
			"🎯 <b>Take Profit 1:</b> Rp%.2f (+1.5%%)\n"+
			"🚀 <b>Take Profit 2:</b> Rp%.2f (+2.5%%)\n"+
			"⚖️ <b>Risk-to-Reward:</b> %s\n\n"+
			"📈 <b>Indikator Intraday:</b>\n"+
			"• RSI (14): %.1f\n"+
			"• MACD Status: %s\n"+
			"• EMA Trend: %s\n"+
			"• Rating TradingView: %s\n\n"+
			"📝 <b>Catatan Scalper Senior:</b>\n<i>%s</i>",
		icon, res.Symbol,
		res.Market,
		res.Recommendation,
		res.EntryZone.Min, res.EntryZone.Max,
		res.StopLoss,
		res.TakeProfit1,
		res.TakeProfit2,
		res.RiskRewardRatio,
		res.TechnicalSummary.RSI14,
		res.TechnicalSummary.MACDStatus,
		res.TechnicalSummary.EMATrend,
		res.TechnicalSummary.TradingViewRating,
		res.TraderNotes,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, cardMsg)
}

// handleAIAdvisorCommand menangani pemindaian + saran AI Provider
func (b *BotDispatcher) handleAIAdvisorCommand(ctx context.Context, chatID string, args []string) {
	if len(args) == 0 {
		reply := "⚠️ <b>Format Perintah AI Salah!</b>\n" +
			"Gunakan format: <code>/ai &lt;SYMBOL&gt;</code> atau <code>/ai &lt;MARKET&gt; &lt;SYMBOL&gt;</code>\n\n" +
			"<b>Contoh:</b> <code>/ai BBCA</code> atau <code>/advise indodax btc_idr</code>"
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, reply)
		return
	}

	market := models.MarketIDX
	symbol := args[0]
	if len(args) >= 2 {
		if strings.EqualFold(args[0], "INDODAX") || strings.EqualFold(args[0], "CRYPTO") {
			market = models.MarketIndodax
		}
		symbol = args[1]
	} else if strings.Contains(strings.ToLower(symbol), "_idr") || strings.EqualFold(symbol, "btc") || strings.EqualFold(symbol, "eth") {
		market = models.MarketIndodax
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("🧠 <b>Menghubungi AI Provider & Analisis Scalper Senior untuk %s (%s)... Mohon tunggu sebentar.</b>", strings.ToUpper(symbol), market))

	req := models.ScanRequest{
		Market:    market,
		Symbol:    symbol,
		Timeframe: "15M",
	}

	scanRes, err := b.signalService.GenerateSignal(ctx, req)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal pemindaian teknikal: %v", err))
		return
	}

	if b.aiAdvisorService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ AI Advisor Service belum diaktifkan.")
		return
	}

	adviceMsg, err := b.aiAdvisorService.GenerateScalpingAdvice(ctx, scanRes)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal membuat saran AI: %v", err))
		return
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, adviceMsg)
}

// handleAskCommand menangani pertanyaan strategi/psikologi pasar bebas dari user ke AI Mentor
func (b *BotDispatcher) handleAskCommand(ctx context.Context, chatID, question string) {
	if question == "" {
		reply := "⚠️ <b>Format Perintah Salah!</b>\n" +
			"Gunakan format: <code>/ask &lt;PERTANYAAN TRADING ANDA&gt;</code>\n\n" +
			"<b>Contoh:</b> <code>/ask bagaimana strategi mengelola posisi saat bursa volatilitas tinggi?</code>"
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, reply)
		return
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "🤖 <b>AI Trading Mentor sedang memikirkan jawaban terbaik... Mohon tunggu.</b>")

	if b.aiAdvisorService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ AI Advisor Service belum diaktifkan.")
		return
	}

	answer, err := b.aiAdvisorService.AnswerTradingQuestion(ctx, question)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mendapatkan jawaban AI: %v", err))
		return
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, answer)
}

// handleIDXCommand mengirim ringkasan data saham IDX
func (b *BotDispatcher) handleIDXCommand(ctx context.Context, chatID string) {
	tickers, err := b.marketService.FetchIDXSummary(ctx)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mengambil data IDX: %v", err))
		return
	}

	sb := strings.Builder{}
	sb.WriteString("📈 <b>Ringkasan Saham Unggulan IDX (Bursa Efek Indonesia):</b>\n")
	sb.WriteString("----------------------------------------\n")

	for i, item := range tickers {
		if i >= 10 {
			break
		}
		sign := "+"
		if item.Change < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("• <b>%s</b>: Rp%.0f (%s%.2f%%)\n", item.Symbol, item.LastPrice, sign, item.Percentage))
	}
	sb.WriteString("\n💡 <i>Gunakan perintah <code>/scalp idx &lt;SYMBOL&gt;</code> atau <code>/ai &lt;SYMBOL&gt;</code> untuk sinyal & saran AI.</i>")

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, sb.String())
}

// handleIndodaxCommand mengirim data ticker Indodax
func (b *BotDispatcher) handleIndodaxCommand(ctx context.Context, chatID string, args []string) {
	pair := "btc_idr"
	if len(args) > 0 {
		pair = args[0]
	}

	ticker, err := b.marketService.FetchIndodaxTicker(ctx, pair)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mengambil data Indodax: %v", err))
		return
	}

	t := ticker.Ticker
	msg := fmt.Sprintf(
		"🪙 <b>Indodax Ticker: %s</b>\n"+
			"----------------------------------------\n"+
			"<b>Harga Terakhir:</b> Rp%s\n"+
			"<b>Tertinggi 24j:</b> Rp%s\n"+
			"<b>Terendah 24j:</b> Rp%s\n"+
			"<b>Volume (IDR):</b> Rp%s\n\n"+
			"💡 <i>Gunakan <code>/scalp indodax %s</code> atau <code>/ai indodax %s</code> untuk saran scalping AI.</i>",
		strings.ToUpper(pair), t.Last, t.High, t.Low, t.VolIDR, pair, pair,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, msg)
}

// handleHistoryCommand mengirimkan 5 riwayat sinyal scalping terbaru dari database SQLite
func (b *BotDispatcher) handleHistoryCommand(ctx context.Context, chatID string) {
	signals, err := b.signalService.GetSignalHistory(ctx, 5)
	if err != nil || len(signals) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "📭 Belum ada riwayat sinyal scalping harian yang tersimpan.")
		return
	}

	sb := strings.Builder{}
	sb.WriteString("📜 <b>Riwayat 5 Sinyal Scalping Terbaru (SQLite):</b>\n")
	sb.WriteString("----------------------------------------\n")

	for i, s := range signals {
		sb.WriteString(fmt.Sprintf("%d. <b>%s (%s)</b>: <u>%s</u> | SL: %.0f | RRR: %s\n", i+1, s.Symbol, s.Market, s.Action, s.StopLoss, s.RiskRewardRatio))
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, sb.String())
}

// handleJournalCommand menampilkan entri jurnal trading terbaru dari SQLite
func (b *BotDispatcher) handleJournalCommand(ctx context.Context, chatID string) {
	if b.journalService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Journal Service belum diaktifkan.")
		return
	}

	journals, err := b.journalService.GetJournalHistory(ctx, 5)
	if err != nil || len(journals) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "📖 Belum ada catatan jurnal trading tersimpan di SQLite.")
		return
	}

	sb := strings.Builder{}
	sb.WriteString("📓 <b>Jurnal Performa Scalping Trading (SQLite):</b>\n")
	sb.WriteString("----------------------------------------\n")

	for i, j := range journals {
		sb.WriteString(fmt.Sprintf("%d. <b>%s (%s)</b> | Status: <u>%s</u>\n   Entri: Rp%.2f | Target: Rp%.2f | SL: Rp%.2f\n   <i>%s</i>\n\n",
			i+1, j.Symbol, j.Market, j.Status, j.EntryPrice, j.TargetPrice, j.StopPrice, j.Notes))
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, sb.String())
}

// handleStatsCommand menampilkan statistik Win Rate (%), Total PnL, dan Profit Factor
func (b *BotDispatcher) handleStatsCommand(ctx context.Context, chatID string) {
	if b.journalService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Journal Service belum diaktifkan.")
		return
	}

	stats, err := b.journalService.GetJournalStats(ctx)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal menghitung statistik: %v", err))
		return
	}

	msg := fmt.Sprintf(
		"📊 <b>STATISTIK ANALISIS PERFORMA SCALPER:</b>\n"+
			"----------------------------------------\n"+
			" Total Rekomendasi Trade: <b>%d</b>\n"+
			"🎯 Trade Menang (Win): <b>%d</b>\n"+
			"🛑 Trade Kalah (Loss): <b>%d</b>\n"+
			"⏳ Posisi Berjalan (Open): <b>%d</b>\n\n"+
			"🏆 <b>Win Rate:</b> <u>%.1f%%</u>\n"+
			"💰 <b>Total Estimasi PnL:</b> Rp%.2f\n"+
			"⚖️ <b>Profit Factor:</b> %.2fx\n\n"+
			"💡 <i>Gunakan disiplin Stop Loss untuk menjaga Win Rate & Profit Factor tetap optimal!</i>",
		stats.TotalTrades, stats.WinCount, stats.LossCount, stats.OpenCount,
		stats.WinRate, stats.TotalPnL, stats.ProfitFactor,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, msg)
}

// handleNewsCommand menangani analisis sentimen berita pasar AI
func (b *BotDispatcher) handleNewsCommand(ctx context.Context, chatID string, args []string) {
	if len(args) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ <b>Gunakan format:</b> <code>/news &lt;SYMBOL&gt;</code>\nContoh: <code>/news BBCA</code>")
		return
	}

	symbol := args[0]
	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("📰 <b>Mengambil berita pasar & penilaian sentimen AI untuk %s...</b>", strings.ToUpper(symbol)))

	if b.newsService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ News Service belum aktif.")
		return
	}

	articles, err := b.newsService.GetNewsSentiment(ctx, symbol)
	if err != nil || len(articles) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mengambil sentimen berita: %v", err))
		return
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("📰 <b>BERITA & SKOR SENTIMEN AI: %s</b>\n----------------------------------------\n", strings.ToUpper(symbol)))

	for i, a := range articles {
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   Sentimen: <u>%s</u> (Skor: %.2f)\n   <i>%s</i>\n\n", i+1, a.Title, a.Sentiment, a.Score, a.Summary))
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, sb.String())
}

// handleBandarCommand menangani analisis ketebalan Bid/Ask dan lonjakan volume
func (b *BotDispatcher) handleBandarCommand(ctx context.Context, chatID string, args []string) {
	if len(args) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ <b>Gunakan format:</b> <code>/bandar &lt;SYMBOL&gt;</code>\nContoh: <code>/bandar btc_idr</code>")
		return
	}

	symbol := args[0]
	market := models.MarketIDX
	if strings.Contains(strings.ToLower(symbol), "_idr") || strings.EqualFold(symbol, "btc") {
		market = models.MarketIndodax
	}

	if b.bandarService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Bandarmologi Service belum aktif.")
		return
	}

	res, err := b.bandarService.AnalyzeBandarmologi(ctx, market, symbol)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal analisis bandarmologi: %v", err))
		return
	}

	msg := fmt.Sprintf(
		"🐋 <b>ANALISIS BANDARMOLOGI & VOLUME SPIKE: %s (%s)</b>\n"+
			"----------------------------------------\n"+
			"<b>Aksi Bandar:</b> <u>%s</u>\n"+
			"⚖️ <b>Bid/Ask Ratio:</b> %.2fx (Bid: %.0f | Ask: %.0f)\n"+
			"⚡ <b>Volume Spike:</b> %.2fx vs Rata-rata 5 Hari\n\n"+
			"📝 <b>Catatan Analis:</b>\n<i>%s</i>",
		res.Symbol, res.Market, res.BandarAction, res.BidAskRatio, res.TotalBidVolume, res.TotalAskVolume, res.VolumeSpikeRatio, res.RecommendationNote,
	)

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, msg)
}

// handleExportCommand mengirimkan file dokumen CSV rekap jurnal trading ke chat Telegram
func (b *BotDispatcher) handleExportCommand(ctx context.Context, chatID string) {
	if b.journalService == nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Journal Service belum aktif.")
		return
	}

	csvData, err := b.journalService.ExportJournalCSV(ctx)
	if err != nil || len(csvData) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mengekspor jurnal: %v", err))
		return
	}

	fileName := fmt.Sprintf("Trading_Journal_Report_%s.csv", time.Now().Format("20060102_1504"))
	err = b.telegramClient.SendDocument(ctx, b.botToken, chatID, fileName, csvData)
	if err != nil {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("❌ Gagal mengirim file dokumen CSV: %v", err))
	} else {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "📑 <b>Dokumen Rekap Jurnal Trading (.CSV) berhasil terkirim!</b>")
	}
}

// handleWatchCommand mengelola watchlist pribadi pengguna
func (b *BotDispatcher) handleWatchCommand(ctx context.Context, chatID string, args []string) {
	if len(args) == 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ <b>Gunakan format:</b>\n• <code>/watch add &lt;SYMBOL&gt;</code>\n• <code>/watch list</code>\n• <code>/watch del &lt;SYMBOL&gt;</code>")
		return
	}

	subCmd := strings.ToLower(args[0])

	switch subCmd {
	case "add":
		if len(args) < 2 {
			_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Masukkan simbol aset: <code>/watch add BBCA</code>")
			return
		}
		symbol := strings.ToUpper(args[1])
		if config.DB != nil {
			var exist models.UserWatchlist
			err := config.DB.Where("chat_id = ? AND symbol = ?", chatID, symbol).First(&exist).Error
			if err != nil {
				config.DB.Create(&models.UserWatchlist{ChatID: chatID, Symbol: symbol, Market: models.MarketIDX})
			}
		}
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("✅ <b>%s</b> berhasil ditambahkan ke Watchlist pribadi Anda!", symbol))

	case "list":
		var list []models.UserWatchlist
		if config.DB != nil {
			config.DB.Where("chat_id = ?", chatID).Find(&list)
		}
		if len(list) == 0 {
			_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "📌 Watchlist pribadi Anda masih kosong.")
			return
		}
		sb := strings.Builder{}
		sb.WriteString("📌 <b>DAFTAR WATCHLIST PRIBADI ANDA:</b>\n----------------------------------------\n")
		for i, w := range list {
			sb.WriteString(fmt.Sprintf("%d. <b>%s</b> (%s)\n", i+1, w.Symbol, w.Market))
		}
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, sb.String())

	case "del":
		if len(args) < 2 {
			_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Masukkan simbol aset: <code>/watch del BBCA</code>")
			return
		}
		symbol := strings.ToUpper(args[1])
		if config.DB != nil {
			config.DB.Where("chat_id = ? AND symbol = ?", chatID, symbol).Delete(&models.UserWatchlist{})
		}
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("🗑️ <b>%s</b> berhasil dihapus dari Watchlist.", symbol))
	}
}

// handleAlertCommand memasang alarm harga kustom per pengguna
func (b *BotDispatcher) handleAlertCommand(ctx context.Context, chatID string, args []string) {
	if len(args) < 2 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ <b>Gunakan format:</b> <code>/alert &lt;SYMBOL&gt; &lt;HARGA_TARGET&gt;</code>\nContoh: <code>/alert BBCA 10200</code>")
		return
	}

	symbol := strings.ToUpper(args[0])
	priceVal, err := strconv.ParseFloat(args[1], 64)
	if err != nil || priceVal <= 0 {
		_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, "⚠️ Target harga tidak valid! Masukkan angka positif.")
		return
	}

	if config.DB != nil {
		alert := models.UserPriceAlert{
			ChatID:      chatID,
			Symbol:      symbol,
			Market:      models.MarketIDX,
			TargetPrice: priceVal,
			Condition:   "ABOVE",
		}
		config.DB.Create(&alert)
	}

	_ = b.telegramClient.SendMessage(ctx, b.botToken, chatID, fmt.Sprintf("🔔 <b>ALARM HARGA DIPASANG!</b>\nBot akan memberi tahu Anda saat harga <b>%s</b> menyentuh Rp%.2f.", symbol, priceVal))
}
