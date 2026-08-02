package services

import (
	"context"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"strings"
	"trading-analysis-bot/clients"
	"trading-analysis-bot/models"
)

// MarketServiceInterface mendefinisikan modul pengelolaan data pasar
type MarketServiceInterface interface {
	FetchIDXSummary(ctx context.Context) ([]models.IDXTickerResponse, error)
	FetchIndodaxTicker(ctx context.Context, pair string) (*models.IndodaxTickerResponse, error)
	FetchIndodaxTopPairs(ctx context.Context) ([]string, error)
	SaveMarketDataCache(ctx context.Context, data *models.MarketData) error
}

// MarketService adalah layanan pengelolaan data pasar saham & kripto
type MarketService struct {
	db            *gorm.DB
	idxClient     clients.IDXClientInterface
	indodaxClient clients.IndodaxClientInterface
}

// NewMarketService menginisialisasi Market Service
func NewMarketService(
	db *gorm.DB,
	idxClient clients.IDXClientInterface,
	indodaxClient clients.IndodaxClientInterface,
) *MarketService {
	return &MarketService{
		db:            db,
		idxClient:     idxClient,
		indodaxClient: indodaxClient,
	}
}

// FetchIDXSummary penarikan data saham IDX
func (s *MarketService) FetchIDXSummary(ctx context.Context) ([]models.IDXTickerResponse, error) {
	tickers, err := s.idxClient.GetTickers(ctx)
	if err != nil {
		return nil, err
	}

	// Simpan secara terisolasi ke database SQLite sebagai data histori cache
	go func(items []models.IDXTickerResponse) {
		for _, item := range items {
			_ = s.SaveMarketDataCache(context.Background(), &models.MarketData{
				Symbol:           item.Symbol,
				Market:           models.MarketIDX,
				LastPrice:        item.LastPrice,
				Volume:           item.Volume,
				PriceChange:      item.Change,
				PriceChangeRatio: item.Percentage,
			})
		}
	}(tickers)

	return tickers, nil
}

// FetchIndodaxTicker penarikan ticker harga Indodax
func (s *MarketService) FetchIndodaxTicker(ctx context.Context, pair string) (*models.IndodaxTickerResponse, error) {
	resp, err := s.indodaxClient.GetTicker(ctx, pair)
	if err != nil {
		return nil, err
	}

	// Simpan cache ke SQLite
	if resp != nil {
		lastVal, _ := strconv.ParseFloat(resp.Ticker.Last, 64)
		highVal, _ := strconv.ParseFloat(resp.Ticker.High, 64)
		lowVal, _ := strconv.ParseFloat(resp.Ticker.Low, 64)
		volVal, _ := strconv.ParseFloat(resp.Ticker.VolIDR, 64)

		_ = s.SaveMarketDataCache(ctx, &models.MarketData{
			Symbol:    pair,
			Market:    models.MarketIndodax,
			LastPrice: lastVal,
			High24h:   highVal,
			Low24h:    lowVal,
			Volume:    volVal,
		})
	}

	return resp, nil
}

// FetchIndodaxTopPairs memfilter koin-koin kripto Indodax ber-volume tinggi yang paling cocok untuk scalping harian
func (s *MarketService) FetchIndodaxTopPairs(ctx context.Context) ([]string, error) {
	summaries, err := s.indodaxClient.GetSummaries(ctx)
	if err != nil || len(summaries) == 0 {
		return []string{"btc_idr", "eth_idr", "sol_idr", "usdt_idr"}, nil
	}

	var liquidPairs []string
	for pair, item := range summaries {
		if !strings.HasSuffix(pair, "_idr") {
			continue
		}
		volVal, _ := strconv.ParseFloat(item.VolIDR, 64)
		if volVal >= 500000000 { // Volume > Rp 500 Juta untuk likuiditas aman
			liquidPairs = append(liquidPairs, pair)
		}
	}

	if len(liquidPairs) == 0 {
		return []string{"btc_idr", "eth_idr", "sol_idr", "usdt_idr"}, nil
	}

	return liquidPairs, nil
}

// SaveMarketDataCache menyimpan entitas MarketData ke SQLite via GORM
func (s *MarketService) SaveMarketDataCache(ctx context.Context, data *models.MarketData) error {
	if s.db == nil {
		return nil
	}
	if err := s.db.WithContext(ctx).Create(data).Error; err != nil {
		return fmt.Errorf("gagal menyimpan cache market data: %w", err)
	}
	return nil
}
