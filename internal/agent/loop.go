package agent

import (
	"context"
	"errors"
	"time"

	"hermes-go/internal/llm"
	"hermes-go/internal/memory"
	"hermes-go/internal/skills"
	"hermes-go/internal/tools"
)

// ErrMaxIterationsReached se retorna cuando el loop llega al limite sin respuesta final.
var ErrMaxIterationsReached = errors.New("max iterations reached without final response")

// LoopOptions configura el ConversationLoop.
type LoopOptions struct {
	LLM        *llm.Client
	Registry   *tools.Registry
	Memory     *memory.Manager
	Skills     *skills.Loader
	Prompt     *PromptBuilder
	MaxIter    int
	ToolBudget int
	Timeout    time.Duration
}

// ConversationLoop ejecuta el ciclo de conversacion para una sesion.
// Fase 6.
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
// Fase 6.
func (l *ConversationLoop) Run(ctx context.Context, sess *Session, userMsg string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	sess.AppendUser(userMsg)

	// Recall de memoria para este turno (usa cache de background)
	var memCtx string
	if l.memory != nil {
		memCtx = l.memory.Prefetch(ctx, userMsg, sess.ID)
		l.memory.QueuePrefetch(ctx, userMsg, sess.ID)
	}

	systemPrompt := l.prompt.Build(sess, memCtx)

	for iter := 0; iter < l.maxIter; iter++ {
		// Fase 6: llamar al LLM
		// resp, err := l.llm.ChatCompletion(ctx, llm.ChatRequest{...})
		panic("not implemented: Phase 6")
	}

	return "", ErrMaxIterationsReached
}
