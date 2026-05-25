package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"hermes-go/internal/config"
	"hermes-go/internal/providers"
)

// ChatRequest es la peticion al LLM.
type ChatRequest struct {
	System   string
	Messages []openai.ChatCompletionMessageParamUnion
	Tools    []openai.ChatCompletionToolParam
	Stream   bool
}

// ChatResponse es la respuesta del LLM.
type ChatResponse struct {
	Content   string
	ToolCalls []openai.ChatCompletionMessageToolCall
	Model     string
	Usage     openai.CompletionUsage
}

// Client envuelve el SDK de OpenAI con retries y clasificacion de errores.
// Fase 3.
type Client struct {
	inner   *openai.Client
	model   string
	timeout time.Duration
	retry   backoff.BackOff
}

// NewClient construye el cliente LLM. Si cfg.Provider esta definido, resuelve
// baseURL y APIKey desde el registro de proveedores. En caso contrario usa
// cfg.BaseURL y cfg.APIKey directamente.
func NewClient(ctx context.Context, cfg config.LLMConfig) (*Client, error) {
	baseURL := cfg.BaseURL
	apiKey := cfg.APIKey
	var extraHeaders map[string]string

	if cfg.Provider != "" {
		resolved, err := providers.Resolve(ctx, cfg.Provider)
		if err != nil {
			return nil, fmt.Errorf("llm: resolver proveedor: %w", err)
		}
		if baseURL == "" {
			baseURL = resolved.BaseURL
		}
		if apiKey == "" {
			apiKey = resolved.APIKey
		}
		extraHeaders = resolved.DefaultHeaders
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.Multiplier = 2
	bo.MaxInterval = 16 * time.Second
	bo.MaxElapsedTime = time.Duration(cfg.MaxRetries) * 16 * time.Second

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}
	for k, v := range extraHeaders {
		opts = append(opts, option.WithHeader(k, v))
	}

	c := openai.NewClient(opts...)
	return &Client{
		inner:   c,
		model:   cfg.Model,
		timeout: cfg.Timeout,
		retry:   bo,
	}, nil
}

// ChatCompletion ejecuta una llamada al LLM con reintentos en errores transitorios.
// Fase 3.
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	panic("not implemented: Phase 3")
}
