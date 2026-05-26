package email

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
)

// ThreadIndex mapea Message-ID y References a un thread ID estable.
// Permite que respuestas al mismo hilo mantengan el mismo SessionID.
// Fase 10.
type ThreadIndex struct {
	mu      sync.RWMutex
	threads map[string]string // messageID -> threadID
}

func NewThreadIndex() *ThreadIndex {
	return &ThreadIndex{threads: make(map[string]string)}
}

// Resolve devuelve el thread ID para un mensaje dado su Message-ID,
// In-Reply-To y References. Si es el primer mensaje del hilo, genera
// un thread ID nuevo basado en el Message-ID.
func (t *ThreadIndex) Resolve(messageID, inReplyTo string, references []string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Revisar si hay un thread existente via In-Reply-To
	if inReplyTo != "" {
		if threadID, ok := t.threads[inReplyTo]; ok {
			if messageID != "" {
				t.threads[messageID] = threadID
			}
			return threadID
		}
	}

	// Revisar la cadena de references (de mas reciente a mas antiguo)
	for i := len(references) - 1; i >= 0; i-- {
		ref := references[i]
		if threadID, ok := t.threads[ref]; ok {
			if messageID != "" {
				t.threads[messageID] = threadID
			}
			return threadID
		}
	}

	// Nuevo hilo: generar ID deterministico desde el Message-ID
	threadID := threadIDFor(messageID)
	if messageID != "" {
		t.threads[messageID] = threadID
	}
	return threadID
}

// threadIDFor genera un ID deterministico desde el Message-ID raiz del hilo.
func threadIDFor(rootMessageID string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(rootMessageID))))
	return fmt.Sprintf("email_%x", h[:8])
}
