package stress_test

import (
	"fmt"
	"sync"
	"testing"

	"hermes-go/internal/memory"
)

// TestConcurrentMemoryWrites verifica que BuiltinProvider soporta
// escrituras concurrentes sin corrupcion de datos.
// Ejecutar con: go test -race -run TestConcurrentMemoryWrites ./test/stress/
func TestConcurrentMemoryWrites(t *testing.T) {
	t.Skip("stress test: ejecutar manualmente con -race. Requiere Fase 2.")

	dir := t.TempDir()
	p := memory.NewBuiltinProvider(dir, 2200, 1375)

	const goroutines = 50
	const writesPerGoroutine = 10

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*writesPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				if _, err := p.Add("memory", fmt.Sprintf("entry goroutine=%d turn=%d", idx, j)); err != nil {
					errs <- fmt.Errorf("goroutine %d write %d: %w", idx, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

// TestMemoryCharLimit verifica que el limite de caracteres se respeta
// cuando se agregan entradas que exceden el maximo.
func TestMemoryCharLimit(t *testing.T) {
	t.Skip("stress test: requiere Fase 2.")
	panic("not implemented: Phase 2 memory test")
}
