package email

import (
	"context"
	"fmt"
	"net/smtp"

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
func (s *SMTPSender) Send(_ context.Context, msg platforms.OutgoingMessage) error {
	subject := "Re: message"
	if msg.Metadata != nil {
		if subj, ok := msg.Metadata["subject"]; ok && subj != "" {
			subject = "Re: " + subj
		}
	}

	inReplyTo := ""
	if msg.Metadata != nil {
		inReplyTo = msg.Metadata["message_id"]
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	headers := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n",
		s.cfg.From, msg.ChatID, subject,
	)
	if inReplyTo != "" {
		headers += fmt.Sprintf("In-Reply-To: <%s>\r\nReferences: <%s>\r\n", inReplyTo, inReplyTo)
	}
	body := headers + "\r\n" + msg.Text

	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{msg.ChatID}, []byte(body)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// SendTyping es no-op para email (no hay indicador de escritura).
func (s *SMTPSender) SendTyping(_ context.Context, _ string) error { return nil }
