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
func (t *stdioTransport) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mcp stdio marshal %s: %w", method, err)
	}
	data = append(data, '\n')

	type writeResult struct{ err error }
	done := make(chan writeResult, 1)
	go func() {
		_, werr := t.stdin.Write(data)
		done <- writeResult{err: werr}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("mcp stdio write %s: %w", method, r.err)
		}
	}

	line, err := t.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("mcp stdio read %s: %w", method, err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("mcp stdio unmarshal %s: %w", method, err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp rpc %s error %d: %s", method, resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// Close mata el subproceso.
func (t *stdioTransport) Close() error {
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	return t.cmd.Wait()
}
