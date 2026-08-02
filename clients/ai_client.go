package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
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
		baseURL = "https://ai.kostan.my.id/v1/chat/completions"
	}
	apiKey := os.Getenv("AI_API_KEY")
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	timeout := 30 * time.Second
	if timeoutStr := os.Getenv("AI_TIMEOUT"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil && parsedTimeout > 0 {
			timeout = parsedTimeout
		}
	}

	return &AIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
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

// normalizeURL memastikan URL endpoint AI Provider memiliki suffix /chat/completions
func normalizeURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "https://api.openai.com/v1/chat/completions"
	}
	// Jika belum berakhiran /chat/completions
	if !strings.HasSuffix(urlStr, "/chat/completions") {
		urlStr = strings.TrimRight(urlStr, "/")
		urlStr += "/chat/completions"
	}
	return urlStr
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
	if err == nil && strings.TrimSpace(res) != "" {
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
		if secErr == nil && strings.TrimSpace(secRes) != "" {
			return secRes, nil
		}
	}

	return "", fmt.Errorf("semua AI Provider mengalami timeout/error: %w", err)
}

// doRequest mengeksekusi HTTP POST request ke AI Provider endpoint
func (c *AIClient) doRequest(ctx context.Context, apiURL, apiKey, modelName, systemPrompt, userPrompt string) (string, error) {
	apiURL = normalizeURL(apiURL)

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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal membaca body respon AI Provider (%s): %w", apiURL, err)
	}

	respStr := strings.TrimSpace(string(respBody))

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("AI Provider (%s) merespons status HTTP %d, raw response: %s", apiURL, resp.StatusCode, respStr)
	}

	var apiResp OpenAIResponsePayload
	contentStr := ""

	if err := json.Unmarshal(respBody, &apiResp); err == nil && len(apiResp.Choices) > 0 {
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			return "", fmt.Errorf("AI Provider Error: %s", apiResp.Error.Message)
		}
		contentStr = apiResp.Choices[0].Message.Content
	} else {
		// Fallback: Periksa apakah respon berformat SSE Stream (data: ...)
		if strings.Contains(respStr, "data:") {
			lines := strings.Split(respStr, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "data:") {
					jsonStr := strings.TrimSpace(line[5:])
					if jsonStr == "[DONE]" || jsonStr == "" {
						continue
					}
					var chunk struct {
						Choices []struct {
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
						} `json:"choices"`
					}
					if err := json.Unmarshal([]byte(jsonStr), &chunk); err == nil {
						if len(chunk.Choices) > 0 {
							contentStr += chunk.Choices[0].Delta.Content
						}
					}
				}
			}
		}

		if contentStr == "" {
			return "", fmt.Errorf("gagal decode respon AI Provider: %v, raw response: %s", err, respStr)
		}
	}

	// Sanitasi opsional jika konten dibungkus markdown codeblock ```json ... ```
	jsonRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	matches := jsonRegex.FindStringSubmatch(contentStr)
	if len(matches) > 1 {
		contentStr = strings.TrimSpace(matches[1])
	}

	return contentStr, nil
}

