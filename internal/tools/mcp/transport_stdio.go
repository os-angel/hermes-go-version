package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// stdioTransport implementa Transport sobre stdin/stdout de un subproceso.
// Un mutex serializa todas las llamadas RPC (igual que el cliente MCP de Hermes Python).
// Fase 13.
type stdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
}

func newStdioTransport(command string, args, env []string) (*stdioTransport, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp stdio start: %w", err)
	}
	return &stdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

// Call envia una solicitud JSON-RPC y espera la respuesta. Serializado por mutex.
// Pendiente - Fase 13.
func (t *stdioTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	panic("not implemented: Phase 13")
}

// Close mata el subproceso.
func (t *stdioTransport) Close() error {
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	return t.cmd.Wait()
}
