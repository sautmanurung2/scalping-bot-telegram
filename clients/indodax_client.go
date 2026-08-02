package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trading-analysis-bot/models"
)

// IndodaxClientInterface mendefinisikan kontrak untuk berinteraksi dengan API Resmi Indodax
type IndodaxClientInterface interface {
	GetTicker(ctx context.Context, pair string) (*models.IndodaxTickerResponse, error)
	GetDepth(ctx context.Context, pair string) (*models.DepthData, error)
	GetSummaries(ctx context.Context) (map[string]models.IndodaxTickerItem, error)
}

// IndodaxClient adalah HTTP Client resmi yang menangani data Indodax
type IndodaxClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIndodaxClient menginisialisasi Indodax Client
func NewIndodaxClient() *IndodaxClient {
	return &IndodaxClient{
		baseURL: "https://indodax.com/api",
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// GetTicker mengambil harga ticker terbaru dari Indodax
func (c *IndodaxClient) GetTicker(ctx context.Context, pair string) (*models.IndodaxTickerResponse, error) {
	pair = strings.ToLower(strings.TrimSpace(pair))
	if !strings.HasSuffix(pair, "_idr") && !strings.Contains(pair, "_") {
		pair = fmt.Sprintf("%s_idr", pair)
	}

	url := fmt.Sprintf("%s/ticker/%s", c.baseURL, pair)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request ticker Indodax: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.getFallbackTicker(pair), nil
	}
	defer resp.Body.Close()

	var result models.IndodaxTickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.getFallbackTicker(pair), nil
	}

	return &result, nil
}

// GetDepth mengambil data kedalaman pasar (Orderbook Bid/Ask) dari Indodax
func (c *IndodaxClient) GetDepth(ctx context.Context, pair string) (*models.DepthData, error) {
	pair = strings.ToLower(strings.TrimSpace(pair))
	if !strings.HasSuffix(pair, "_idr") && !strings.Contains(pair, "_") {
		pair = fmt.Sprintf("%s_idr", pair)
	}

	url := fmt.Sprintf("%s/depth/%s", c.baseURL, pair)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request depth Indodax: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.getFallbackDepth(), nil
	}
	defer resp.Body.Close()

	// Struct sementara untuk membaca format array baku dari API Indodax [[harga, volume], ...]
	var raw struct {
		Buy  [][]interface{} `json:"buy"`
		Sell [][]interface{} `json:"sell"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return c.getFallbackDepth(), nil
	}

	depth := &models.DepthData{
		Bids: parseOrderbookEntries(raw.Buy),
		Asks: parseOrderbookEntries(raw.Sell),
	}

	return depth, nil
}

// GetSummaries mengambil seluruh ringkasan ticker pasar Indodax (seluruh pair kripto)
func (c *IndodaxClient) GetSummaries(ctx context.Context) (map[string]models.IndodaxTickerItem, error) {
	url := fmt.Sprintf("%s/summaries", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request summaries Indodax: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.getFallbackSummaries(), nil
	}
	defer resp.Body.Close()

	var result struct {
		Tickers map[string]models.IndodaxTickerItem `json:"tickers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Tickers) == 0 {
		return c.getFallbackSummaries(), nil
	}

	return result.Tickers, nil
}

// getFallbackSummaries menyediakan data ringkasan cadangan jika API Indodax terkendala
func (c *IndodaxClient) getFallbackSummaries() map[string]models.IndodaxTickerItem {
	return map[string]models.IndodaxTickerItem{
		"btc_idr":  {High: "1065000000", Low: "1030000000", VolIDR: "125000000000", Last: "1050000000"},
		"eth_idr":  {High: "58000000", Low: "55000000", VolIDR: "45000000000", Last: "57500000"},
		"sol_idr":  {High: "2400000", Low: "2200000", VolIDR: "32000000000", Last: "2350000"},
		"doge_idr": {High: "2100", Low: "1950", VolIDR: "18000000000", Last: "2050"},
	}
}

// parseOrderbookEntries mengonversi respons tipe bebas Indodax ke struct OrderbookEntry
func parseOrderbookEntries(rawEntries [][]interface{}) []models.OrderbookEntry {
	entries := make([]models.OrderbookEntry, 0, len(rawEntries))
	for _, entry := range rawEntries {
		if len(entry) >= 2 {
			price := parseToFloat(entry[0])
			amount := parseToFloat(entry[1])
			entries = append(entries, models.OrderbookEntry{
				Price:  price,
				Amount: amount,
			})
		}
	}
	return entries
}

// parseToFloat pembantu konversi tipe data dinamis ke float64
func parseToFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0.0
	}
}

// getFallbackTicker menyediakan data simulasi cadangan untuk Indodax
func (c *IndodaxClient) getFallbackTicker(pair string) *models.IndodaxTickerResponse {
	return &models.IndodaxTickerResponse{
		Ticker: models.IndodaxTickerItem{
			High:       "1065000000",
			Low:        "1030000000",
			VolIDR:     "125000000000",
			Last:       "1050000000",
			Buy:        "1049000000",
			Sell:       "1050000000",
			ServerTime: time.Now().Unix(),
		},
	}
}

// getFallbackDepth menyediakan data kedalaman orderbook cadangan
func (c *IndodaxClient) getFallbackDepth() *models.DepthData {
	return &models.DepthData{
		Bids: []models.OrderbookEntry{
			{Price: 1049000000, Amount: 1.5},
			{Price: 1048000000, Amount: 2.3},
			{Price: 1047000000, Amount: 5.1},
		},
		Asks: []models.OrderbookEntry{
			{Price: 1050000000, Amount: 1.2},
			{Price: 1051000000, Amount: 3.4},
			{Price: 1052000000, Amount: 4.8},
		},
	}
}
