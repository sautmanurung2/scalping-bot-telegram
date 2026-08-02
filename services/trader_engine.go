package services

import (
	"fmt"
	"math"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// TraderEngineInterface mendefinisikan modul analisis pasar berstandar Senior Daily Scalper Trader 15+ Tahun
type TraderEngineInterface interface {
	AnalyzeMarket(
		req models.ScanRequest,
		lastPrice float64,
		tvResult *clients.TradingViewTechnicalResult,
		depth *models.DepthData,
	) *models.ScanResponse
}

// TraderEngine adalah implementasi logika keputusan trading scalping harian profesional
type TraderEngine struct{}

// NewTraderEngine menginisialisasi modul analisis trader senior
func NewTraderEngine() *TraderEngine {
	return &TraderEngine{}
}

// AnalyzeMarket mengeksekusi algoritma rekomendasi intraday scalping harian
func (e *TraderEngine) AnalyzeMarket(
	req models.ScanRequest,
	lastPrice float64,
	tvResult *clients.TradingViewTechnicalResult,
	depth *models.DepthData,
) *models.ScanResponse {
	// 1. Kalkulasi Trend & Momentum Intraday Fast Moving Average (EMA50 & EMA200)
	isUptrend := lastPrice >= tvResult.EMA200 || tvResult.EMA50 >= tvResult.EMA200
	rsiScore := tvResult.RSI14
	isBullishMACD := tvResult.MACDValue > tvResult.MACDSignal

	// 2. Analisis Kedalaman Orderbook Intraday (Bid vs Ask ratio)
	bidAskRatio := 1.0
	if depth != nil && len(depth.Bids) > 0 && len(depth.Asks) > 0 {
		totalBidVol := 0.0
		totalAskVol := 0.0
		for _, b := range depth.Bids {
			totalBidVol += b.Amount
		}
		for _, a := range depth.Asks {
			totalAskVol += a.Amount
		}
		if totalAskVol > 0 {
			bidAskRatio = totalBidVol / totalAskVol
		}
	}

	// 3. Kalkulasi Presisi Scalping Harian: Entry Zone, Tight Stop Loss (-1%), dan Fast Take Profits (+1.2% & +2.2%)
	entryMin := math.Floor(lastPrice * 0.995)
	entryMax := math.Ceil(lastPrice * 1.002)

	// Stop Loss Ketat Scalping (0.95% s/d 1.1% dari last price)
	stopLoss := math.Floor(lastPrice * 0.99)
	riskAmount := lastPrice - stopLoss
	if riskAmount <= 0 {
		riskAmount = lastPrice * 0.01
		stopLoss = lastPrice - riskAmount
	}

	// Fast Scalping Profits
	takeProfit1 := math.Ceil(lastPrice + (1.5 * riskAmount)) // ~+1.5%
	takeProfit2 := math.Ceil(lastPrice + (2.5 * riskAmount)) // ~+2.5%

	// Risk-to-Reward Ratio (RRR)
	rrrValue := (takeProfit2 - lastPrice) / riskAmount
	rrrStr := fmt.Sprintf("1:%.2f", rrrValue)

	// 4. Penentuan Aksi Rekomendasi Scalper Harian
	action := models.ActionHold
	emaTrendStr := "BELOW_EMA_200 (BEARISH INTRADAY)"
	if isUptrend {
		emaTrendStr = "ABOVE_EMA_200 (BULLISH INTRADAY)"
	}

	macdStatusStr := "BEARISH_CROSS"
	if isBullishMACD {
		macdStatusStr = "BULLISH_CROSS"
	}

	var notes string

	if isUptrend && rsiScore >= 48 && rsiScore <= 65 && isBullishMACD && bidAskRatio >= 1.25 {
		action = models.ActionStrongBuy
		notes = fmt.Sprintf(
			"⚡ FAST SCALP STRONG BUY: %s menunjukkan momentum intraday yang sangat kuat di atas EMA 200. RSI (%.1f) berada dalam zona akselerasi sehat. Tekanan Beli Orderbook (Ratio Bid/Ask: %.2f) mengindikasikan lonjakan volume harian. Target TP1 di Rp%.0f (+1.5%%), Wajib Stop Loss di Rp%.0f (-1.0%%). No Overnight!",
			req.Symbol, rsiScore, bidAskRatio, takeProfit1, stopLoss,
		)
	} else if isUptrend && (isBullishMACD || rsiScore > 50) {
		action = models.ActionBuy
		notes = fmt.Sprintf(
			"✅ DAILY SCALP BUY: %s berada dalam tren intraday naik. Pasang Limit Order bertahap di area Entry Zone (Rp%.0f - Rp%.0f) dengan Stop Loss ketat di Rp%.0f.",
			req.Symbol, entryMin, entryMax, stopLoss,
		)
	} else if rsiScore > 72 {
		action = models.ActionSell
		notes = fmt.Sprintf(
			"⚠️ OVERBOUGHT SCALP (TAKE PROFIT): RSI (%.1f) menyentuh zona jenuh beli intraday. Segera amankan profit sebelum terjadi koreksi cepat harian.",
			rsiScore,
		)
	} else if rsiScore < 32 && !isUptrend {
		action = models.ActionAvoid
		notes = fmt.Sprintf(
			"⛔ AVOID SCALPING: Aset %s berada dalam tekanan jual intraday kuat (Downtrend Extreme). Hindari entri sebelum terjadi struktur konfirmasi Reversal 15M.",
			req.Symbol,
		)
	} else {
		action = models.ActionHold
		notes = fmt.Sprintf(
			"⏳ WAIT & SEE (SIDEWAYS): %s sedang dalam fase konsolidasi intraday. Belum ada kepastian arah breakout untuk scalping harian.",
			req.Symbol,
		)
	}

	return &models.ScanResponse{
		Symbol:          req.Symbol,
		Market:          req.Market,
		Recommendation: action,
		EntryZone:       models.EntryZone{Min: entryMin, Max: entryMax},
		StopLoss:        stopLoss,
		TakeProfit1:     takeProfit1,
		TakeProfit2:     takeProfit2,
		RiskRewardRatio: rrrStr,
		TechnicalSummary: models.TechSummary{
			RSI14:             rsiScore,
			MACDStatus:        macdStatusStr,
			EMATrend:          emaTrendStr,
			TradingViewRating: tvResult.TechnicalRating,
		},
		TraderNotes: notes,
	}
}
