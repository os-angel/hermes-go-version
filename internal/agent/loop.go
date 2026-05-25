package agent

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"hermes-go/internal/llm"
	"hermes-go/internal/memory"
	"hermes-go/internal/tools"
)

// ErrMaxIterationsReached se retorna cuando el loop llega al limite sin respuesta final.
var ErrMaxIterationsReached = errors.New("max iterations reached without final response")

// LoopOptions configura el ConversationLoop.
type LoopOptions struct {
	LLM        *llm.Client
	Registry   *tools.Registry
	Memory     *memory.Manager
	Prompt     *PromptBuilder
	MaxIter    int
	ToolBudget int
	Timeout    time.Duration
}

// ConversationLoop ejecuta el ciclo de conversacion para una sesion.
type ConversationLoop struct {
	llm      *llm.Client
	executor *ToolExecutor
	memory   *memory.Manager
	prompt   *PromptBuilder
	maxIter  int
	budget   int
	timeout  time.Duration
}

func NewConversationLoop(opts LoopOptions) *ConversationLoop {
	if opts.MaxIter <= 0 {
		opts.MaxIter = 12
	}
	if opts.ToolBudget <= 0 {
		opts.ToolBudget = 60000
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	return &ConversationLoop{
		llm:      opts.LLM,
		executor: NewToolExecutor(opts.Registry),
		memory:   opts.Memory,
		prompt:   opts.Prompt,
		maxIter:  opts.MaxIter,
		budget:   opts.ToolBudget,
		timeout:  opts.Timeout,
	}
}

// Run procesa un turno completo: mensaje -> LLM -> tools -> respuesta final.
// Bloquea hasta que el LLM responde sin tool calls, se agota MaxIter, o ctx vence.
func (l *ConversationLoop) Run(ctx context.Context, sess *Session, userMsg string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	sess.AppendUser(userMsg)

	// Prefetch de memoria para este turno
	var memCtx string
	if l.memory != nil {
		memCtx = l.memory.Prefetch(ctx, userMsg, sess.ID)
		l.memory.QueuePrefetch(ctx, userMsg, sess.ID)
	}

	systemPrompt := l.prompt.Build(sess, memCtx)

	// Obtener schemas de tools del registry
	var toolSchemas []map[string]any
	if l.executor.registry != nil {
		toolSchemas = l.executor.registry.Schemas()
	}

	for iter := 0; iter < l.maxIter; iter++ {
		start := time.Now()
		resp, err := l.llm.ChatCompletion(ctx, llm.ChatRequest{
			System:   systemPrompt,
			Messages: sess.Snapshot(),
			Tools:    toolSchemas,
		})
		if err != nil {
			return "", err
		}

		slog.Debug("llm response",
			"session_id", sess.ID,
			"iter", iter+1,
			"tool_calls", len(resp.ToolCalls),
			"content_len", len(resp.Content),
			"duration_ms", time.Since(start).Milliseconds(),
		)

		// Si no hay tool calls, la respuesta es final
		if len(resp.ToolCalls) == 0 {
			sess.AppendAssistant(resp.Content)
			if l.memory != nil {
				l.memory.SyncTurn(ctx, userMsg, resp.Content, sess.ID)
			}
			return resp.Content, nil
		}

		// Hay tool calls: agregarlos al historial y ejecutarlos
		sess.AppendAssistantWithCalls(resp.Content, resp.ToolCalls)
		results := l.executor.Execute(ctx, resp.ToolCalls, l.budget)
		for _, r := range results {
			content := r.Content
			if r.Err != nil {
				content = `{"error": "` + r.Err.Error() + `"}`
			}
			sess.AppendToolResult(r.ToolCallID, content)
		}
	}

	return "", ErrMaxIterationsReached
}
