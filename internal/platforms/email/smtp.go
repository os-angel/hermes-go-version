package email

import (
	"context"

	"hermes-go/internal/platforms"
)

// SMTPSender implementa platforms.Sender enviando via SMTP.
// Fase 10.
type SMTPSender struct {
	cfg SMTPConfig
}

// SMTPConfig contiene la configuracion de conexion SMTP.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Name() string { return "email" }

// Send envia un mensaje via SMTP.
// El campo ChatID se interpreta como la direccion de destino (To).
// Pendiente - Fase 10.
func (s *SMTPSender) Send(ctx context.Context, msg platforms.OutgoingMessage) error {
	panic("not implemented: Phase 10")
}

// SendTyping es no-op para email (no hay indicador de escritura).
func (s *SMTPSender) SendTyping(_ context.Context, _ string) error { return nil }
