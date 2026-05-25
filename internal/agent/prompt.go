package agent

import (
	"fmt"
	"strings"
	"time"

	"hermes-go/internal/memory"
	"hermes-go/internal/skills"
)

// PromptBuilder construye el system prompt en 3 capas.
// Fase 5.
type PromptBuilder struct {
	identity     string
	toolGuidance string
	skills       *skills.Loader
	memory       *memory.Manager
	modelName    string
	timezone     *time.Location
}

type PromptOptions struct {
	Identity  string
	Skills    *skills.Loader
	Memory    *memory.Manager
	Model     string
	Timezone  *time.Location
}

func NewPromptBuilder(opts PromptOptions) *PromptBuilder {
	tz := opts.Timezone
	if tz == nil {
		tz = time.UTC
	}
	return &PromptBuilder{
		identity:  opts.Identity,
		skills:    opts.Skills,
		memory:    opts.Memory,
		modelName: opts.Model,
		timezone:  tz,
	}
}

// Build retorna el system prompt completo (stable + context + volatile).
// Fase 5.
func (p *PromptBuilder) Build(sess *Session, memoryContext string) string {
	stable, context, volatile := p.BuildLayers(sess, memoryContext)
	parts := []string{}
	for _, part := range []string{stable, context, volatile} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n")
}

// BuildLayers retorna las 3 capas separadas.
// Fase 5.
func (p *PromptBuilder) BuildLayers(sess *Session, memoryContext string) (stable, context, volatile string) {
	// stable: identidad + tool guidance + skills index + memory snapshot (frozen al inicio de sesion)
	var stableParts []string
	if p.identity != "" {
		stableParts = append(stableParts, p.identity)
	}
	if p.skills != nil {
		if block := p.skills.BuildSystemPromptBlock(); block != "" {
			stableParts = append(stableParts, block)
		}
	}
	if p.memory != nil {
		if block := p.memory.BuildSystemPrompt(); block != "" {
			stableParts = append(stableParts, block)
		}
	}
	stable = strings.Join(stableParts, "\n\n")

	// context: info del canal/plataforma + sender
	if sess != nil {
		context = fmt.Sprintf("Plataforma: %s\nSesion: %s", sess.Platform, sess.ID)
	}

	// volatile: timestamp + modelo activo + memory context recall
	var volatileParts []string
	volatileParts = append(volatileParts, fmt.Sprintf("Fecha y hora: %s", time.Now().In(p.timezone).Format(time.RFC3339)))
	if p.modelName != "" {
		volatileParts = append(volatileParts, fmt.Sprintf("Modelo: %s", p.modelName))
	}
	if memoryContext != "" {
		volatileParts = append(volatileParts, memoryContext)
	}
	volatile = strings.Join(volatileParts, "\n")

	return
}
