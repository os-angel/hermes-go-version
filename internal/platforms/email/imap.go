package email

import (
	"context"
	"log/slog"
	"time"

	"hermes-go/internal/platforms"
)

// IMAPPoller conecta via IMAP IDLE y emite IncomingMessage por cada correo nuevo.
// Fase 10.
type IMAPPoller struct {
	cfg      IMAPConfig
	out      chan<- platforms.IncomingMessage
	interval time.Duration
}

// IMAPConfig contiene la configuracion de conexion IMAP.
type IMAPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Mailbox  string
	TLS      bool
}

func NewIMAPPoller(cfg IMAPConfig, out chan<- platforms.IncomingMessage) *IMAPPoller {
	return &IMAPPoller{cfg: cfg, out: out, interval: 30 * time.Second}
}

// Start corre el loop de polling IMAP hasta ctx.Done().
// Pendiente - Fase 10.
func (p *IMAPPoller) Start(ctx context.Context) error {
	slog.Info("imap poller start", "host", p.cfg.Host, "mailbox", p.cfg.Mailbox)
	panic("not implemented: Phase 10")
}
