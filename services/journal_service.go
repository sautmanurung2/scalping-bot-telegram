package services

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"trading-analysis-bot/models"
)

// JournalStats DTO ringkasan statistik performa trading
type JournalStats struct {
	TotalTrades  int     `json:"total_trades"`
	WinCount     int     `json:"win_count"`
	LossCount    int     `json:"loss_count"`
	OpenCount    int     `json:"open_count"`
	WinRate      float64 `json:"win_rate"`
	TotalPnL     float64 `json:"total_pnl"`
	ProfitFactor float64 `json:"profit_factor"`
}

// JournalServiceInterface mendefinisikan kontrak pengelola jurnal trading
type JournalServiceInterface interface {
	CreateJournalFromSignal(ctx context.Context, signal *models.ScanResponse) (*models.TradingJournal, error)
	GetJournalHistory(ctx context.Context, limit int) ([]models.TradingJournal, error)
	GetJournalStats(ctx context.Context) (*JournalStats, error)
	ExportJournalCSV(ctx context.Context) ([]byte, error)
}

// JournalService menangani logika bisnis pencatatan & analisis jurnal trading
type JournalService struct {
	db *gorm.DB
}

// NewJournalService menginisialisasi JournalService
func NewJournalService(db *gorm.DB) *JournalService {
	return &JournalService{db: db}
}

// CreateJournalFromSignal mencatat rekomendasi sinyal ke dalam jurnal SQLite
func (s *JournalService) CreateJournalFromSignal(ctx context.Context, signal *models.ScanResponse) (*models.TradingJournal, error) {
	if s.db == nil || signal == nil {
		return nil, fmt.Errorf("database atau data sinyal tidak valid")
	}

	entryPrice := (signal.EntryZone.Min + signal.EntryZone.Max) / 2.0
	journal := models.TradingJournal{
		Symbol:      signal.Symbol,
		Market:      signal.Market,
		EntryPrice:  entryPrice,
		TargetPrice: signal.TakeProfit1,
		StopPrice:   signal.StopLoss,
		Status:      "OPEN",
		Notes:       fmt.Sprintf("Sinyal: %s | RSI: %.1f | RRR: %s", signal.Recommendation, signal.TechnicalSummary.RSI14, signal.RiskRewardRatio),
	}

	if err := s.db.WithContext(ctx).Create(&journal).Error; err != nil {
		return nil, fmt.Errorf("gagal menyimpa jurnal: %w", err)
	}

	return &journal, nil
}

// GetJournalHistory mengambil entri riwayat jurnal trading terbaru
func (s *JournalService) GetJournalHistory(ctx context.Context, limit int) ([]models.TradingJournal, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database belum diinisialisasi")
	}

	var journals []models.TradingJournal
	err := s.db.WithContext(ctx).Order("id desc").Limit(limit).Find(&journals).Error
	return journals, err
}

// GetJournalStats menghitung metrik performa Win Rate, Total PnL, dan Profit Factor
func (s *JournalService) GetJournalStats(ctx context.Context) (*JournalStats, error) {
	if s.db == nil {
		return &JournalStats{}, nil
	}

	var journals []models.TradingJournal
	if err := s.db.WithContext(ctx).Find(&journals).Error; err != nil {
		return nil, err
	}

	stats := &JournalStats{}
	stats.TotalTrades = len(journals)

	var grossProfit, grossLoss float64

	for _, j := range journals {
		switch strings.ToUpper(j.Status) {
		case "WIN_TP1", "WIN_TP2", "WIN":
			stats.WinCount++
			grossProfit += j.ProfitLoss
			stats.TotalPnL += j.ProfitLoss
		case "HIT_SL", "LOSS":
			stats.LossCount++
			grossLoss += absFloat(j.ProfitLoss)
			stats.TotalPnL += j.ProfitLoss
		default:
			stats.OpenCount++
		}
	}

	closedCount := stats.WinCount + stats.LossCount
	if closedCount > 0 {
		stats.WinRate = (float64(stats.WinCount) / float64(closedCount)) * 100.0
	}

	if grossLoss > 0 {
		stats.ProfitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		stats.ProfitFactor = grossProfit // Tak terhingga / 0 loss
	}

	return stats, nil
}

// ExportJournalCSV menghasilkan data CSV dari seluruh riwayat jurnal trading SQLite
func (s *JournalService) ExportJournalCSV(ctx context.Context) ([]byte, error) {
	journals, err := s.GetJournalHistory(ctx, 1000)
	if err != nil {
		return nil, err
	}

	sb := strings.Builder{}
	sb.WriteString("ID,Symbol,Market,EntryPrice,TargetPrice,StopPrice,Status,ProfitLoss,Notes,CreatedAt\n")

	for _, j := range journals {
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%.2f,%.2f,%.2f,%s,%.2f,\"%s\",%s\n",
			j.ID, j.Symbol, j.Market, j.EntryPrice, j.TargetPrice, j.StopPrice, j.Status, j.ProfitLoss, strings.ReplaceAll(j.Notes, "\"", "'"), j.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return []byte(sb.String()), nil
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
