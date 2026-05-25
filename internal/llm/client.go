package llm

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"hermes-go/internal/config"
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

func NewClient(cfg config.LLMConfig) *Client {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.Multiplier = 2
	bo.MaxInterval = 16 * time.Second
	bo.MaxElapsedTime = time.Duration(cfg.MaxRetries) * 16 * time.Second

	c := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)

	return &Client{
		inner:   c,
		model:   cfg.Model,
		timeout: cfg.Timeout,
		retry:   bo,
	}
}

// ChatCompletion ejecuta una llamada al LLM con reintentos en errores transitorios.
// Fase 3.
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	panic("not implemented: Phase 3")
}
