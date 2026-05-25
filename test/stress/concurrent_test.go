package stress_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"hermes-go/internal/agent"
	"hermes-go/internal/config"
)

// TestConcurrentSessions verifica que el SessionCache soporta
// N sesiones simultaneas sin race conditions.
// Ejecutar con: go test -race -run TestConcurrentSessions ./test/stress/
func TestConcurrentSessions(t *testing.T) {
	t.Skip("stress test: ejecutar manualmente con -race")

	store := agent.NewFileSessionStore(t.TempDir())
	cache, err := agent.NewSessionCache(agent.SessionCacheOptions{
		MaxSize: 512,
		TTL:     5 * time.Minute,
		Store:   store,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Shutdown(context.Background())

	const numSessions = 500
	const messagesPerSession = 20

	var wg sync.WaitGroup
	errs := make(chan error, numSessions*messagesPerSession)

	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sessionID := config.SessionsDir() + fmt.Sprintf("_stress_%d", idx)
			ctx := context.Background()
			for j := 0; j < messagesPerSession; j++ {
				sess, err := cache.GetOrCreate(ctx, sessionID, "stress")
				if err != nil {
					errs <- fmt.Errorf("session %d msg %d: %w", idx, j, err)
					return
				}
				sess.AppendUser(fmt.Sprintf("mensaje %d", j))
				sess.AppendAssistant(fmt.Sprintf("respuesta %d", j))
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestRouterWorkerPool verifica que el PlatformRouter distribuye mensajes
// correctamente entre N workers sin perdidas.
func TestRouterWorkerPool(t *testing.T) {
	t.Skip("stress test: ejecutar manualmente")

	panic("not implemented: Phase 8 stress test")
}
