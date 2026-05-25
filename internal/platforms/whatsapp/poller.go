package whatsapp

import (
	"context"
	"log/slog"
	"time"

	"hermes-go/internal/identity"
	"hermes-go/internal/platforms"
)

// Poller hace polling a GET /messages del bridge y normaliza los mensajes.
// Fase 9.
type Poller struct {
	client   *Client
	interval time.Duration
	out      chan<- platforms.IncomingMessage
	id       *Identity
}

func NewPoller(client *Client, out chan<- platforms.IncomingMessage, id *Identity, interval time.Duration) *Poller {
	if interval <= 0 {
		interval = time.Second
	}
	return &Poller{client: client, interval: interval, out: out, id: id}
}

// Start corre el loop de polling hasta ctx.Done().
func (p *Poller) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msgs, err := p.client.Poll(ctx)
			if err != nil {
				slog.Warn("whatsapp poll error", "err", err)
				continue
			}
			for _, m := range msgs {
				if incoming, ok := p.normalize(m); ok {
					select {
					case p.out <- incoming:
					case <-ctx.Done():
						return nil
					}
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Poller) normalize(m bridgeMessage) (platforms.IncomingMessage, bool) {
	if m.Body == "" && !m.HasMedia {
		return platforms.IncomingMessage{}, false
	}
	chatID := m.ChatID
	if p.id != nil {
		chatID = p.id.Canonical(chatID)
	}
	sessID := identity.SessionID("whatsapp", chatID)
	return platforms.IncomingMessage{
		Platform:   "whatsapp",
		SessionID:  sessID,
		ChatID:     m.ChatID, // raw para responder
		SenderID:   m.SenderID,
		SenderName: m.SenderName,
		IsGroup:    m.IsGroup,
		Text:       m.Body,
		MessageID:  m.MessageID,
		ReplyTo:    m.QuotedMessageID,
		ReceivedAt: time.Unix(m.Timestamp/1000, 0),
	}, true
}
