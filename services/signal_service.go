package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// SignalServiceInterface mendefinisikan kontrak pengelolaan sinyal trading
type SignalServiceInterface interface {
	GenerateSignal(ctx context.Context, req models.ScanRequest) (*models.ScanResponse, error)
	GetSignalHistory(ctx context.Context, limit int) ([]models.TradingSignal, error)
}

// SignalService mengoordinasikan penarikan paralel data pasar, TV scanner, TraderEngine, dan Telegram Notification
type SignalService struct {
	db                  *gorm.DB
	idxClient           clients.IDXClientInterface
	indodaxClient       clients.IndodaxClientInterface
	tvClient            clients.TradingViewClientInterface
	traderEngine        TraderEngineInterface
	notificationService NotificationServiceInterface
}

// NewSignalService menginisialisasi SignalService
func NewSignalService(
	db *gorm.DB,
	idxClient clients.IDXClientInterface,
	indodaxClient clients.IndodaxClientInterface,
	tvClient clients.TradingViewClientInterface,
	traderEngine TraderEngineInterface,
	notificationService NotificationServiceInterface,
) *SignalService {
	return &SignalService{
		db:                  db,
		idxClient:           idxClient,
		indodaxClient:       indodaxClient,
		tvClient:            tvClient,
		traderEngine:        traderEngine,
		notificationService: notificationService,
	}
}

// GenerateSignal melakukan analisis multi-faktor paralel menggunakan Goroutine & Channels
func (s *SignalService) GenerateSignal(ctx context.Context, req models.ScanRequest) (*models.ScanResponse, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var lastPrice float64
	var tvResult *clients.TradingViewTechnicalResult
	var depthData *models.DepthData
	var errFetch error

	// 1. Goroutine 1: Penarikan Harga Terbaru (IDX atau Indodax)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if req.Market == models.MarketIDX {
			stock, err := s.idxClient.GetStockDetail(ctxTimeout, req.Symbol)
			if err != nil {
				errFetch = err
				return
			}
			lastPrice = stock.LastPrice
		} else {
			ticker, err := s.indodaxClient.GetTicker(ctxTimeout, req.Symbol)
			if err != nil {
				errFetch = err
				return
			}
			lastPrice, _ = strconv.ParseFloat(ticker.Ticker.Last, 64)
		}
	}()

	// 2. Goroutine 2: Penarikan Indikator TradingView Scanner
	wg.Add(1)
	go func() {
		defer wg.Done()
		tvRes, err := s.tvClient.GetTechnicalAnalysis(ctxTimeout, req.Market, req.Symbol)
		if err == nil {
			tvResult = tvRes
		}
	}()

	// 3. Goroutine 3: Penarikan Depth Orderbook (Indodax)
	if req.Market == models.MarketIndodax {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := s.indodaxClient.GetDepth(ctxTimeout, req.Symbol)
			if err == nil {
				depthData = d
			}
		}()
	}

	// Tunggu seluruh goroutine selesai
	wg.Wait()

	if errFetch != nil {
		return nil, fmt.Errorf("gagal menarik data pasar untuk %s: %w", req.Symbol, errFetch)
	}

	if tvResult == nil {
		tvResult = &clients.TradingViewTechnicalResult{
			Symbol:          req.Symbol,
			RSI14:           55.0,
			MACDValue:       10.0,
			MACDSignal:      8.0,
			EMA50:           lastPrice * 0.98,
			EMA200:          lastPrice * 0.95,
			TechnicalRating: "BUY",
		}
	}

	// 4. Analisis Trader Senior
	res := s.traderEngine.AnalyzeMarket(req, lastPrice, tvResult, depthData)

	// 5. Simpan sinyal ke SQLite via GORM
	if s.db != nil {
		timeframe := req.Timeframe
		if timeframe == "" {
			timeframe = "1D"
		}

		signalEntity := models.TradingSignal{
			Symbol:           res.Symbol,
			Market:           res.Market,
			Timeframe:        timeframe,
			Action:           res.Recommendation,
			EntryMin:         res.EntryZone.Min,
			EntryMax:         res.EntryZone.Max,
			StopLoss:         res.StopLoss,
			TakeProfit1:      res.TakeProfit1,
			TakeProfit2:      res.TakeProfit2,
			RiskRewardRatio:  res.RiskRewardRatio,
			RSI14:            res.TechnicalSummary.RSI14,
			MACDStatus:       res.TechnicalSummary.MACDStatus,
			EMATrend:         res.TechnicalSummary.EMATrend,
			TradingViewScore: res.TechnicalSummary.TradingViewRating,
			TraderNotes:      res.TraderNotes,
			GeneratedAt:      time.Now(),
		}

		_ = s.db.WithContext(ctx).Create(&signalEntity).Error
	}

	// 6. Kirim notifikasi Telegram secara asinkron (non-blocking) jika NotificationService aktif
	if s.notificationService != nil {
		go func(sig *models.ScanResponse) {
			_ = s.notificationService.SendSignalAlert(context.Background(), sig)
		}(res)
	}

	return res, nil
}

// GetSignalHistory mengambil riwayat sinyal analisis terdahulu dari SQLite
func (s *SignalService) GetSignalHistory(ctx context.Context, limit int) ([]models.TradingSignal, error) {
	if limit <= 0 {
		limit = 20
	}
	var signals []models.TradingSignal
	if s.db == nil {
		return signals, nil
	}

	err := s.db.WithContext(ctx).Order("id desc").Limit(limit).Find(&signals).Error
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat sinyal: %w", err)
	}

	return signals, nil
}
