package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// TelegramUser DTO pengirim pesan
type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// TelegramChat DTO chat room
type TelegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

// TelegramMessage DTO struktur pesan masuk
type TelegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from"`
	Chat      TelegramChat  `json:"chat"`
	Date      int64         `json:"date"`
	Text      string        `json:"text"`
}

// TelegramUpdate DTO struktur event Telegram getUpdates
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message"`
}

// TelegramClientInterface mendefinisikan kontrak pemanggilan Telegram Bot API
type TelegramClientInterface interface {
	SendMessage(ctx context.Context, botToken, chatID, text string) error
	SendDocument(ctx context.Context, botToken, chatID, fileName string, fileBytes []byte) error
	GetUpdates(ctx context.Context, botToken string, offset int64, timeout int) ([]TelegramUpdate, error)
}

// TelegramClient adalah HTTP client ke Telegram Bot API
type TelegramClient struct {
	httpClient *http.Client
}

// NewTelegramClient menginisialisasi TelegramClient
func NewTelegramClient() *TelegramClient {
	return &TelegramClient{
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // Timeout aman untuk Long Polling (30s)
		},
	}
}

// SendMessagePayload merepresentasikan payload request JSON ke Telegram API
type SendMessagePayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// SendMessage mengirimkan pesan balasan ke Telegram Chat / Channel
func (c *TelegramClient) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	if botToken == "" || chatID == "" {
		return fmt.Errorf("telegram bot token atau chat ID belum dikonfigurasi")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := SendMessagePayload{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal payload telegram: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("gagal membuat request telegram: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal menghubungi telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Ok          bool   `json:"ok"`
			ErrorCode   int    `json:"error_code"`
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, apiErr.Description)
	}

	return nil
}

// SendDocument mengirimkan berkas dokumen (seperti file CSV) ke Telegram chat
func (c *TelegramClient) SendDocument(ctx context.Context, botToken, chatID, fileName string, fileBytes []byte) error {
	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if botToken == "" || chatID == "" {
		return fmt.Errorf("telegram bot token atau chat ID belum dikonfigurasi")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	_ = w.WriteField("chat_id", chatID)
	fw, err := w.CreateFormFile("document", fileName)
	if err != nil {
		return err
	}
	_, _ = fw.Write(fileBytes)
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetUpdates mengambil daftar pesan/event baru menggunakan Long-Polling
func (c *TelegramClient) GetUpdates(ctx context.Context, botToken string, offset int64, timeout int) ([]TelegramUpdate, error) {
	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	if botToken == "" {
		return nil, fmt.Errorf("telegram bot token tidak boleh kosong")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d", botToken, offset, timeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request getUpdates: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal memanggil getUpdates: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Ok          bool             `json:"ok"`
		Result      []TelegramUpdate `json:"result"`
		Description string           `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gagal decode getUpdates response: %w", err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("telegram API error: %s", result.Description)
	}

	return result.Result, nil
}
