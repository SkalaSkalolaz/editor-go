package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// --- Constants & Configuration ---

const (

DefaultSystemPrompt = `You are a powerful AI assistant for a code editor with LLM support.
**Core Principles:**
1. **Coding**: Create accurate, efficient, and well-structured code in Go language.
2. **Context Understanding**: Analyze the current file, code context, and user tasks.
3. **Editor Integration**: Consider the current line, selected fragment, and project files when generating code.
4. **Safety**: Refuse requests that may be malicious or disrupt editor functionality.
5. **Structure**: Use Markdown for explanations and comments. Always provide explanations for code blocks.
6. **Compatibility**: Ensure your code follows Go standards and can be successfully compiled.
7. **Project Work**: When necessary, refer to other project files to ensure full functionality.

**Additional Instructions:**
- If the user provides a code snippet, analyze it and suggest improvements.
- If the request is unclear, ask clarifying questions.
- Provide only the code that is needed by the user.
- Do not add unnecessary explanations if they are not required.`

	CodeCompletionSystemPrompt = `You are a code completion engine. Your ONLY job is to complete code.

STRICT RULES:
1. Return ONLY the code that continues from the cursor position
2. Do NOT repeat any existing code
3. Do NOT include explanations, markdown, or backticks
4. If you need to explain something, use ONLY code comments
5. Match the existing code style exactly
6. Keep completions concise and relevant
7. If unsure, provide the most likely completion based on context`

	// Таймаут для HTTP клиента
	defaultTimeout = 120 * time.Second

    // Короткий таймаут для inline completion
    completionTimeout = 30 * time.Second
)

// Shared HTTP client to avoid socket exhaustion
var httpClient = &http.Client{
	Timeout: defaultTimeout,
}

// --- Public API ---

// SendMessageToLLM — основная точка входа
// timeout = 0 означает использовать defaultTimeout
func SendMessageToLLM(prompt, providerName, model, apiKey string) (string, error) {
    return SendMessageToLLMWithTimeout(prompt, providerName, model, apiKey, defaultTimeout)
}

// SendMessageToLLMWithTimeout — версия с кастомным таймаутом
func SendMessageToLLMWithTimeout(prompt, providerName, model, apiKey string, timeout time.Duration) (string, error) {
    if timeout <= 0 {
        timeout = defaultTimeout
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    history := []Message{
        {Role: "user", Content: prompt},
    }

    provider, err := newProvider(providerName, model, apiKey)
    if err != nil {
        return "", fmt.Errorf("provider error: %w", err)
    }

    return provider.Send(ctx, history, nil)
}

// --- Internal Types ---

// Message — внутренняя структура для представления сообщений (аналог domain.Message)
type Message struct {
	Role    string
	Content string
}

// Provider — интерфейс для абстракции различных LLM API
type Provider interface {
	Send(ctx context.Context, history []Message, images []string) (string, error)
}

// --- Provider Factory ---

func newProvider(name, model, key string) (Provider, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	switch normalizedName {
	case "ollama":
		return &OllamaProvider{Model: model}, nil
	case "pollinations":
		return &PollinationsProvider{Model: model, Key: key}, nil
	case "openrouter":
		return &OpenRouterProvider{Model: model, Key: key}, nil
	default:
		// Если имя похоже на URL, используем Generic провайдер
		if isURL(name) {
			return &GenericURLProvider{Endpoint: name, Model: model, Key: key}, nil
		}
		// Fallback to Ollama if unknown (или можно возвращать ошибку)
		return &OllamaProvider{Model: model}, nil
	}
}

// --- Provider Implementations ---

// 1. Ollama Provider
type OllamaProvider struct{ Model string }

func (p *OllamaProvider) Send(ctx context.Context, history []Message, images []string) (string, error) {
	url := "http://localhost:11434/v1/chat/completions"
	
	// Ollama обычно не требует системного промпта в body, если он задан в Modelfile,
	// но мы передаем пустой или дефолтный, если нужно переопределить.
	msgs := messagesToMaps(history, images, "") 
	
	payload := map[string]interface{}{
		"model":    p.Model,
		"messages": msgs,
		"stream":   false,
	}
	
	respBody, err := postJSON(ctx, url, payload, "")
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

// 2. Pollinations Provider
type PollinationsProvider struct{ Model, Key string }

func (p *PollinationsProvider) Send(ctx context.Context, history []Message, images []string) (string, error) {
	// Используем HTTPS endpoint, как в рабочем примере
	url := "https://gen.pollinations.ai/v1/chat/completions"
	// Альтернативный URL из примера для справки: "https://text.pollinations.ai/openai"
	sysPrompt := "You are a helpful assistant."

	msgs := messagesToMaps(history, images, sysPrompt)

	payload := map[string]interface{}{
		"model":    p.Model,
		"messages": msgs,
		"seed":     42, // Добавлен seed для детерминированности (из примера)
	}

	// Pollinations часто работает бесплатно без ключа, но если ключ передан, отправляем его.
	respBody, err := postJSON(ctx, url, payload, p.Key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

// 3. OpenRouter Provider
type OpenRouterProvider struct{ Model, Key string }

func (p *OpenRouterProvider) Send(ctx context.Context, history []Message, images []string) (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"
	
	msgs := messagesToMaps(history, images, DefaultSystemPrompt)
	
	payload := map[string]interface{}{
		"model":    p.Model,
		"messages": msgs,
	}

	respBody, err := postJSON(ctx, url, payload, p.Key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

// 4. Generic URL Provider (Custom Endpoint)
type GenericURLProvider struct{ Endpoint, Model, Key string }

func (p *GenericURLProvider) Send(ctx context.Context, history []Message, images []string) (string, error) {
	msgs := messagesToMaps(history, images, DefaultSystemPrompt)
	
	payload := map[string]interface{}{
		"model":    p.Model,
		"messages": msgs,
	}
	
	respBody, err := postJSON(ctx, p.Endpoint, payload, p.Key)
	if err != nil {
		return "", err
	}
	return extractContent(respBody)
}

// --- Helper Functions (Logic) ---

// messagesToMaps конвертирует []Message в формат JSON OpenAI API.
// Добавляет системный промпт и картинки (если есть).
func messagesToMaps(history []Message, images []string, systemPrompt string) []map[string]interface{} {
	msgs := make([]map[string]interface{}, 0, len(history)+1)

	// 1. System Prompt (если задан)
	if systemPrompt != "" {
		msgs = append(msgs, map[string]interface{}{"role": "system", "content": systemPrompt})
	}

	for i, m := range history {
		// Логика для картинок (OpenAI Vision Format)
		// Прикрепляем картинки только к последнему сообщению пользователя
		if i == len(history)-1 && m.Role == "user" && len(images) > 0 {
			msgs = append(msgs, map[string]interface{}{
				"role":    m.Role,
				"content": buildMessageContent(m.Content, images),
			})
		} else {
			msgs = append(msgs, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}
	
	return msgs
}

// buildMessageContent формирует контент. Если есть картинки — возвращает массив объектов.
func buildMessageContent(content string, images []string) interface{} {
	if len(images) == 0 {
		return content
	}

	contentParts := []map[string]interface{}{
		{
			"type": "text",
			"text": content,
		},
	}

	for _, imgBase64 := range images {
		contentParts = append(contentParts, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{
				"url": imgBase64, 
			},
		})
	}
	return contentParts
}

func isURL(s string) bool {
	// Простая проверка, начинается ли строка с http/https
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// --- Helper Functions (Network & Parsing) ---

func postJSON(ctx context.Context, url string, payload interface{}, key string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	// Доп. заголовки для OpenRouter, чтобы они знали источник (опционально)
	if strings.Contains(url, "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/go-gnome-editor")
		req.Header.Set("X-Title", "Go Gnome Editor")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBytes))
	}
	
	return respBytes, nil
}

// extractContent парсит ответ. Сначала пробует стандартный JSON OpenAI формата,
// затем ищет JSON внутри Markdown блоков (для "грязных" ответов).
func extractContent(body []byte) (string, error) {
	return extractContentFromPossibleJSON(string(body))
}

func extractContentFromPossibleJSON(s string) (string, error) {
	s = strings.TrimSpace(s)

	// Структура для покрытия большинства форматов (OpenAI, Ollama, generic)
	type GenericResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Content string `json:"content"` // Некоторые API возвращают контент прямо здесь
			Text    string `json:"text"`    // Legacy completion format
		} `json:"choices"`
		Content string `json:"content"` // Top-level content
		Text    string `json:"text"`    // Top-level text
		Output  string `json:"output"`  // Replicate style
		Error   string `json:"error"`   // Simple error check
	}

	// 1. Попытка распарсить как валидный JSON
	var r GenericResp
	if err := json.Unmarshal([]byte(s), &r); err == nil {
		if r.Error != "" {
			return "", errors.New(r.Error)
		}
		if len(r.Choices) > 0 {
			if r.Choices[0].Message.Content != "" { return r.Choices[0].Message.Content, nil }
			if r.Choices[0].Content != "" { return r.Choices[0].Content, nil }
			if r.Choices[0].Text != "" { return r.Choices[0].Text, nil }
		}
		if r.Content != "" { return r.Content, nil }
		if r.Text != "" { return r.Text, nil }
		if r.Output != "" { return r.Output, nil }
	}
	
	// 2. Если JSON невалиден или пуст, проверяем, не вернула ли модель JSON внутри Markdown.
	// (Частая проблема слабых моделей, которые вместо чистого JSON пишут: "Вот ваш JSON: ```json ... ```")
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		// Рекурсивная попытка распарсить содержимое code block
		if content, err := extractContentFromPossibleJSON(m[1]); err == nil {
			return content, nil
		}
		// Если внутри не JSON, возвращаем просто текст блока
		return m[1], nil
	}

	// 3. Если это не JSON и не Markdown block, но строка не пустая и не похожа на ошибку структуры
	if len(s) > 0 && !strings.HasPrefix(s, "{") {
		// Считаем, что API вернуло raw text
		return s, nil
	}
	
	return "", fmt.Errorf("failed to extract content from response")
}
