package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// AIClientInterface mendefinisikan kontrak pemanggilan API AI Provider
type AIClientInterface interface {
	GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// AIClient adalah REST Client serbaguna yang kompatibel dengan API OpenAI / Gemini / LocalAI
type AIClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAIClient menginisialisasi AIClient dari Environment Variables
func NewAIClient() *AIClient {
	baseURL := os.Getenv("AI_PROVIDER_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/chat/completions"
	}
	apiKey := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &AIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

// OpenAIRequestPayload DTO payload standar Chat Completions
type OpenAIRequestPayload struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// OpenAIMessage DTO struktur pesan instruksi prompt
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponsePayload DTO balasan dari AI Provider API
type OpenAIResponsePayload struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateCompletion mengirimkan request prompt ke AI Provider dengan mekanisme Multi-Provider Circuit Breaker
func (c *AIClient) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.apiKey == "" {
		c.apiKey = os.Getenv("AI_API_KEY")
	}

	if c.apiKey == "" {
		return "", fmt.Errorf("AI_API_KEY belum dikonfigurasi di file .env")
	}

	// Coba provider utama terlebih dahulu
	res, err := c.doRequest(ctx, c.baseURL, c.apiKey, c.model, systemPrompt, userPrompt)
	if err == nil && res != "" {
		return res, nil
	}

	// Circuit Breaker: Jika provider utama gagal, coba Secondary AI Provider jika dikonfigurasi
	secondaryURL := os.Getenv("SECONDARY_AI_PROVIDER_URL")
	secondaryKey := os.Getenv("SECONDARY_AI_API_KEY")
	secondaryModel := os.Getenv("SECONDARY_AI_MODEL")

	if secondaryURL != "" && secondaryKey != "" {
		if secondaryModel == "" {
			secondaryModel = c.model
		}
		secRes, secErr := c.doRequest(ctx, secondaryURL, secondaryKey, secondaryModel, systemPrompt, userPrompt)
		if secErr == nil && secRes != "" {
			return secRes, nil
		}
	}

	return "", fmt.Errorf("semua AI Provider mengalami timeout/error: %w", err)
}

// doRequest mengeksekusi HTTP POST request ke AI Provider endpoint
func (c *AIClient) doRequest(ctx context.Context, apiURL, apiKey, modelName, systemPrompt, userPrompt string) (string, error) {
	payload := OpenAIRequestPayload{
		Model: modelName,
		Messages: []OpenAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   800,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gagal marshal payload AI client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("gagal membuat request AI API: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal menghubungi AI Provider API (%s): %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("AI Provider (%s) merespons status HTTP %d", apiURL, resp.StatusCode)
	}

	var apiResp OpenAIResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("gagal decode respon AI Provider: %w", err)
	}

	if apiResp.Error != nil && apiResp.Error.Message != "" {
		return "", fmt.Errorf("AI Provider Error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("AI Provider tidak mengembalikan respons teks")
	}

	return apiResp.Choices[0].Message.Content, nil
}
