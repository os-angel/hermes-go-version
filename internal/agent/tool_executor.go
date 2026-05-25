package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/openai/openai-go"
	"golang.org/x/sync/errgroup"

	"hermes-go/internal/tools"
)

const MaxConcurrentTools = 8

// ToolResult es el resultado de ejecutar una tool.
type ToolResult struct {
	ToolCallID string
	Content    string
	Err        error
}

// ToolExecutor ejecuta tool calls con concurrencia controlada.
// Fase 6.
type ToolExecutor struct {
	registry *tools.Registry
}

func NewToolExecutor(reg *tools.Registry) *ToolExecutor {
	return &ToolExecutor{registry: reg}
}

// Execute corre las tool calls. Tools con Parallel=true van en goroutines,
// las demas secuencialmente. Respeta budget de caracteres de output.
// Fase 6.
func (e *ToolExecutor) Execute(ctx context.Context, calls []openai.ChatCompletionMessageToolCall, budget int) []ToolResult {
	results := make([]ToolResult, len(calls))

	var parallel, sequential []int
	for i, call := range calls {
		t := e.registry.Get(call.Function.Name)
		if t != nil && t.Parallel {
			parallel = append(parallel, i)
		} else {
			sequential = append(sequential, i)
		}
	}

	// ejecutar paralelos con errgroup + semaforo
	sem := make(chan struct{}, MaxConcurrentTools)
	var mu sync.Mutex
	totalChars := 0

	eg, egCtx := errgroup.WithContext(ctx)
	for _, idx := range parallel {
		i := idx
		call := calls[i]
		eg.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := e.executeOne(egCtx, call)
			mu.Lock()
			totalChars += len(result)
			if totalChars > budget {
				result = result[:max(0, budget-totalChars+len(result))] + "...[truncated]"
			}
			results[i] = ToolResult{ToolCallID: call.ID, Content: result, Err: err}
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	// ejecutar secuenciales
	for _, i := range sequential {
		call := calls[i]
		result, err := e.executeOne(ctx, call)
		totalChars += len(result)
		if totalChars > budget {
			result = result[:max(0, budget-totalChars+len(result))] + "...[truncated]"
		}
		results[i] = ToolResult{ToolCallID: call.ID, Content: result, Err: err}
	}

	return results
}

func (e *ToolExecutor) executeOne(ctx context.Context, call openai.ChatCompletionMessageToolCall) (string, error) {
	var args map[string]any
	// TODO Fase 6: parsear call.Function.Arguments JSON -> args
	_ = args
	result, err := e.registry.Execute(ctx, call.Function.Name, args)
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error()), nil
	}
	return result, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
