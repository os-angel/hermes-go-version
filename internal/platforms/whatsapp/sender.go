package whatsapp

import (
	"context"
	"strings"

	"hermes-go/internal/platforms"
)

const maxChunkSize = 4096

// Sender implementa platforms.Sender para WhatsApp via bridge.
// Fase 9.
type Sender struct {
	client *Client
}

func NewSender(client *Client) *Sender {
	return &Sender{client: client}
}

func (s *Sender) Name() string { return "whatsapp" }

func (s *Sender) Send(ctx context.Context, msg platforms.OutgoingMessage) error {
	if msg.Typing {
		_ = s.client.Typing(ctx, msg.ChatID)
	}

	chunks := splitMessage(msg.Text, maxChunkSize)
	for i, chunk := range chunks {
		replyTo := ""
		if i == 0 {
			replyTo = msg.ReplyTo
		}
		if err := s.client.Send(ctx, msg.ChatID, chunk, replyTo); err != nil {
			return err
		}
	}

	for _, att := range msg.Media {
		if err := s.client.SendMedia(ctx, msg.ChatID, att.LocalPath, att.MimeType, "", att.Filename); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sender) SendTyping(ctx context.Context, chatID string) error {
	return s.client.Typing(ctx, chatID)
}

// splitMessage divide texto largo respetando saltos de linea.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}
	var chunks []string
	for len(text) > maxLen {
		idx := strings.LastIndexByte(text[:maxLen], '\n')
		if idx <= 0 {
			idx = maxLen
		}
		chunks = append(chunks, text[:idx])
		text = strings.TrimLeft(text[idx:], "\n")
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}
