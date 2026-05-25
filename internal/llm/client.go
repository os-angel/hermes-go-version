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
	// Tools en formato map[string]any (output de tools.Registry.Schemas()).
	// Internamente se convierten a openai.ChatCompletionToolParam.
	Tools []map[string]any
}

// ChatResponse es la respuesta del LLM.
type ChatResponse struct {
	Content   string
	ToolCalls []openai.ChatCompletionMessageToolCall
	Model     string
	Usage     openai.CompletionUsage
}

// Client envuelve el SDK de OpenAI con retries y clasificacion de errores.
type Client struct {
	inner   *openai.Client
	model   string
	timeout time.Duration
	maxRetries int
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

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}
	for k, v := range extraHeaders {
		opts = append(opts, option.WithHeader(k, v))
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &Client{
		inner:      openai.NewClient(opts...),
		model:      cfg.Model,
		timeout:    timeout,
		maxRetries: maxRetries,
	}, nil
}

// mapsToToolParams convierte el formato map[string]any del registry al tipo del SDK.
func mapsToToolParams(schemas []map[string]any) []openai.ChatCompletionToolParam {
	out := make([]openai.ChatCompletionToolParam, 0, len(schemas))
	for _, s := range schemas {
		fn, ok := s["function"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		params, _ := fn["parameters"].(map[string]any)

		out = append(out, openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.F(name),
				Description: openai.F(desc),
				Parameters:  openai.F(openai.FunctionParameters(params)),
			}),
		})
	}
	return out
}

// ChatCompletion ejecuta una llamada al LLM con reintentos en errores transitorios.
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Construir lista de mensajes: system primero, luego el historial
	var messages []openai.ChatCompletionMessageParamUnion
	if req.System != "" {
		messages = append(messages, openai.SystemMessage(req.System))
	}
	messages = append(messages, req.Messages...)

	params := openai.ChatCompletionNewParams{
		Model:    openai.F(openai.ChatModel(c.model)),
		Messages: openai.F(messages),
	}
	if len(req.Tools) > 0 {
		params.Tools = openai.F(mapsToToolParams(req.Tools))
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Second
	bo.Multiplier = 2
	bo.MaxInterval = 16 * time.Second
	bo.MaxElapsedTime = time.Duration(c.maxRetries) * 16 * time.Second

	var resp *openai.ChatCompletion
	var lastErr error

	attempt := 0
	for {
		attempt++
		r, err := c.inner.Chat.Completions.New(ctx, params)
		if err == nil {
			resp = r
			break
		}
		lastErr = err

		class := Classify(err)
		if class != Transient {
			return ChatResponse{}, fmt.Errorf("llm: %w", err)
		}

		wait := bo.NextBackOff()
		if wait == backoff.Stop || attempt >= c.maxRetries {
			return ChatResponse{}, fmt.Errorf("llm: max retries (%d) alcanzados: %w", c.maxRetries, lastErr)
		}

		select {
		case <-ctx.Done():
			return ChatResponse{}, ctx.Err()
		case <-time.After(wait):
		}
	}

	if len(resp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("llm: respuesta vacia (sin choices)")
	}

	choice := resp.Choices[0]
	result := ChatResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Model:     resp.Model,
		Usage:     resp.Usage,
	}
	return result, nil
}
