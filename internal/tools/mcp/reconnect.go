package mcp

import (
	"context"
	"log/slog"
	"time"
)

const (
	reconnectInitial = time.Second
	reconnectMax     = 60 * time.Second
)

// reconnectLoop intenta reconectar llamando a connect() con backoff exponencial
// hasta que ctx sea cancelado o connect() retorne nil.
// Sigue el patron del MCP client de Hermes Python.
func reconnectLoop(ctx context.Context, serverName string, connect func() error) {
	delay := reconnectInitial
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := connect()
		if err == nil {
			delay = reconnectInitial
			return
		}
		slog.Warn("mcp reconnect failed", "server", serverName, "err", err, "retry_in", delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, reconnectMax)
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
