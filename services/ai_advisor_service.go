package services

import (
	"context"
	"fmt"
	"log"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// AIAdvisorServiceInterface mendefinisikan kontrak layanan saran trading AI
type AIAdvisorServiceInterface interface {
	GenerateScalpingAdvice(ctx context.Context, scanRes *models.ScanResponse) (string, error)
	AnswerTradingQuestion(ctx context.Context, question string) (string, error)
}

// AIAdvisorService mengolah data pasar & teknikal menjadi saran scalping AI
type AIAdvisorService struct {
	aiClient clients.AIClientInterface
}

// NewAIAdvisorService menginisialisasi AIAdvisorService
func NewAIAdvisorService(aiClient clients.AIClientInterface) *AIAdvisorService {
	return &AIAdvisorService{
		aiClient: aiClient,
	}
}

const seniorScalperSystemPrompt = `You are a Senior Financial Market Intraday Scalper and Quantitative Trader with 15+ years of experience trading Indonesian Stock Exchange (IDX) and Crypto markets (Indodax).
Your sole focus is DAILY INTRADAY SCALPING (5M/15M timeframes).
Your core directives:
1. CAPITAL PROTECTION FIRST: Always prioritize tight Stop Losses (-0.8% to -1.2%) and realistic Take Profit targets (+1.0% to +2.5%).
2. ZERO FOMO: Warn traders strictly against chasing green candles (>2% gain in 15 mins). Recommend Limit Orders near EMA support.
3. CONCISE & ACTIONABLE: Return output in clean HTML format suitable for Telegram messages.
4. LANGUAGE: Always respond in INDONESIAN (Bahasa Indonesia).`

// GenerateScalpingAdvice menghasilkan kartu saran trading AI khusus scalping harian
func (s *AIAdvisorService) GenerateScalpingAdvice(ctx context.Context, scanRes *models.ScanResponse) (string, error) {
	if scanRes == nil {
		return "", fmt.Errorf("data sinyal tidak boleh kosong")
	}

	userPrompt := fmt.Sprintf(
		"Berikan saran trading scalping harian (intraday) untuk aset berikut:\n"+
			"- Simbol: %s (Pasar: %s)\n"+
			"- Rekomendasi Sinyal Teknikal: %s\n"+
			"- Rentang Entry: %.2f - %.2f\n"+
			"- Stop Loss: %.2f\n"+
			"- Target TP1: %.2f | TP2: %.2f\n"+
			"- Risk-Reward Ratio: %s\n"+
			"- Indikator: RSI(14)=%.1f, MACD=%s, Trend=%s, Rating TV=%s\n"+
			"- Catatan Trader: %s\n\n"+
			"Susun saran dalam 4 bagian singkat:\n"+
			"1. ⚡ Bias Scalping Harian (Bullish/Bearish/Sideways & Tingkat Keyakinan %%)\n"+
			"2. 💡 Saran Eksekusi Entry (Limit Order / Retest EMA / Hindari FOMO)\n"+
			"3. 🛡️ Manajemen Risiko Scalping (Saran Stop Loss & Sizing Modal)\n"+
			"4. ⚠️ Peringatan Intraday (Aturan jam tutup bursa / slippage orderbook)",
		scanRes.Symbol, scanRes.Market,
		scanRes.Recommendation,
		scanRes.EntryZone.Min, scanRes.EntryZone.Max,
		scanRes.StopLoss,
		scanRes.TakeProfit1, scanRes.TakeProfit2,
		scanRes.RiskRewardRatio,
		scanRes.TechnicalSummary.RSI14,
		scanRes.TechnicalSummary.MACDStatus,
		scanRes.TechnicalSummary.EMATrend,
		scanRes.TechnicalSummary.TradingViewRating,
		scanRes.TraderNotes,
	)

	advice, err := s.aiClient.GenerateCompletion(ctx, seniorScalperSystemPrompt, userPrompt)
	if err != nil {
		log.Printf("⚠️ [AI ADVISOR ERROR] Gagal menghasilkan saran AI: %v", err)
		// Fallback cerdas jika AI API offline atau token belum dikonfigurasi
		return fmt.Sprintf(
			"⚡ <b>AI DAILY SCALPING ADVICE (FALLBACK ENGINE): %s (%s)</b>\n"+
				"----------------------------------------\n"+
				"📊 <b>Bias Scalping:</b> %s (Intraday Momentum)\n\n"+
				"💡 <b>Saran Eksekusi:</b>\n"+
				"• Beli secara bertahap pada rentang <b>Rp%.2f - Rp%.2f</b>.\n"+
				"• Gunakan <i>Limit Order</i> di dekat area support EMA. Hindari mengejar harga yang melonjak tinggi.\n\n"+
				"🛡️ <b>Manajemen Risiko Scalper:</b>\n"+
				"• Pasang Stop Loss disiplin di level <b>Rp%.2f</b> (Maksimal risiko 1%% modal).\n"+
				"• Target TP1 di <b>Rp%.2f</b> (+1.2%%) dan TP2 di <b>Rp%.2f</b> (+2.2%%).\n\n"+
				"⚠️ <b>Peringatan Intraday:</b>\n"+
				"• <i>No Overnight Rule:</i> Wajib amankan profit/tutup posisi sebelum bursa tutup harian (15:50 WIB).\n\n"+
				"📌 <b>Detail Error API AI:</b> <code>%v</code>",
			scanRes.Symbol, scanRes.Market,
			scanRes.Recommendation,
			scanRes.EntryZone.Min, scanRes.EntryZone.Max,
			scanRes.StopLoss,
			scanRes.TakeProfit1, scanRes.TakeProfit2,
			err,
		), nil
	}

	header := fmt.Sprintf("🧠 <b>AI SENIOR SCALPER ADVICE: %s (%s)</b>\n----------------------------------------\n", scanRes.Symbol, scanRes.Market)
	return header + advice, nil
}

// AnswerTradingQuestion merespons pertanyaan strategi/psikologi pasar pengguna
func (s *AIAdvisorService) AnswerTradingQuestion(ctx context.Context, question string) (string, error) {
	if question == "" {
		return "", fmt.Errorf("pertanyaan tidak boleh kosong")
	}

	userPrompt := fmt.Sprintf("Pertanyaan Trader: '%s'\n\nBerikan jawaban singkat, logis, praktis, dan mendidik berstandar Senior Scalping Trader (maksimal 3 paragraf).", question)

	answer, err := s.aiClient.GenerateCompletion(ctx, seniorScalperSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("AI Provider tidak dapat merespons saat ini. Pastikan AI_API_KEY di file .env sudah benar: %w", err)
	}

	header := fmt.Sprintf("💬 <b>KONSULTASI AI TRADING MENTOR</b>\n<i>Pertanyaan: \"%s\"</i>\n----------------------------------------\n", question)
	return header + answer, nil
}
