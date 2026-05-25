package agent

import (
	"sync"
	"time"

	"github.com/openai/openai-go"
)

// Session representa el estado de una conversacion activa.
type Session struct {
	ID        string
	Platform  string
	Messages  []openai.ChatCompletionMessageParamUnion
	CreatedAt time.Time
	LastUsed  time.Time
	Metadata  map[string]string
	mu        sync.RWMutex
}

func NewSession(id, platform string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		Platform:  platform,
		CreatedAt: now,
		LastUsed:  now,
		Metadata:  make(map[string]string),
	}
}

// AppendUser agrega un mensaje del usuario. Thread-safe.
func (s *Session) AppendUser(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, openai.UserMessage(text))
}

// AppendAssistant agrega una respuesta del asistente. Thread-safe.
func (s *Session) AppendAssistant(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, openai.AssistantMessage(text))
}

// AppendAssistantWithCalls agrega un mensaje de asistente que incluye tool calls. Thread-safe.
func (s *Session) AppendAssistantWithCalls(content string, calls []openai.ChatCompletionMessageToolCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	callParams := make([]openai.ChatCompletionMessageToolCallParam, len(calls))
	for i, tc := range calls {
		callParams[i] = openai.ChatCompletionMessageToolCallParam{
			ID:   openai.F(tc.ID),
			Type: openai.F(openai.ChatCompletionMessageToolCallType(tc.Type)),
			Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      openai.F(tc.Function.Name),
				Arguments: openai.F(tc.Function.Arguments),
			}),
		}
	}
	msg := openai.ChatCompletionAssistantMessageParam{
		Role:      openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
		ToolCalls: openai.F(callParams),
	}
	if content != "" {
		msg.Content = openai.F([]openai.ChatCompletionAssistantMessageParamContentUnion{
			openai.TextPart(content),
		})
	}
	s.Messages = append(s.Messages, msg)
}

// AppendToolResult agrega el resultado de una tool call. Thread-safe.
func (s *Session) AppendToolResult(toolCallID, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, openai.ToolMessage(toolCallID, content))
}

// Snapshot retorna una copia inmutable del historial (para LLM calls).
func (s *Session) Snapshot() []openai.ChatCompletionMessageParamUnion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]openai.ChatCompletionMessageParamUnion, len(s.Messages))
	copy(cp, s.Messages)
	return cp
}

// Touch actualiza LastUsed.
func (s *Session) Touch() {
	s.mu.Lock()
	s.LastUsed = time.Now()
	s.mu.Unlock()
}

// Reset borra el historial pero mantiene metadata.
func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = nil
}
