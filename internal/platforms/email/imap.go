package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"hermes-go/internal/platforms"
)

// IMAPPoller conecta via IMAP y emite IncomingMessage por cada correo nuevo.
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
func (p *IMAPPoller) Start(ctx context.Context) error {
	slog.Info("imap poller start", "host", p.cfg.Host, "mailbox", p.cfg.Mailbox)

	addr := fmt.Sprintf("%s:%d", p.cfg.Host, p.cfg.Port)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Poll inmediato al arrancar
	if err := p.poll(ctx, addr); err != nil {
		slog.Error("imap initial poll", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.poll(ctx, addr); err != nil {
				slog.Error("imap poll", "err", err)
			}
		}
	}
}

// poll abre una conexion IMAP, busca mensajes no leidos y los emite.
func (p *IMAPPoller) poll(ctx context.Context, addr string) error {
	options := &imapclient.Options{}
	if p.cfg.TLS {
		options.TLSConfig = &tls.Config{ServerName: p.cfg.Host}
	}

	var c *imapclient.Client
	var err error
	if p.cfg.TLS {
		c, err = imapclient.DialTLS(addr, options)
	} else {
		c, err = imapclient.DialInsecure(addr, options)
	}
	if err != nil {
		return fmt.Errorf("imap dial: %w", err)
	}
	defer c.Close()

	if err := c.Login(p.cfg.Username, p.cfg.Password).Wait(); err != nil {
		return fmt.Errorf("imap login: %w", err)
	}

	if _, err := c.Select(p.cfg.Mailbox, nil).Wait(); err != nil {
		return fmt.Errorf("imap select: %w", err)
	}

	searchData, err := c.Search(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		return fmt.Errorf("imap search: %w", err)
	}
	if searchData == nil {
		return nil
	}
	seqSet, ok := searchData.All.(imap.SeqSet)
	if !ok || len(seqSet) == 0 {
		return nil
	}

	fetchCmd := c.Fetch(searchData.All, &imap.FetchOptions{
		Flags:    true,
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{},
		},
	})
	if fetchCmd == nil {
		return nil
	}
	defer fetchCmd.Close()

	for {
		msgData := fetchCmd.Next()
		if msgData == nil {
			break
		}
		buf, err := msgData.Collect()
		if err != nil {
			slog.Warn("imap collect message", "err", err)
			continue
		}
		if err := p.processMsg(ctx, buf); err != nil {
			slog.Warn("imap process message", "err", err)
		}
	}
	return nil
}

// processMsg convierte un FetchMessageBuffer en IncomingMessage y lo emite al canal.
func (p *IMAPPoller) processMsg(ctx context.Context, buf *imapclient.FetchMessageBuffer) error {
	if buf.Envelope == nil {
		return nil
	}

	from := ""
	if len(buf.Envelope.From) > 0 {
		addr := buf.Envelope.From[0]
		from = addr.Mailbox + "@" + addr.Host
	}
	if from == "" || from == p.cfg.Username {
		return nil // Ignorar mensajes propios (anti-loop)
	}

	subject := buf.Envelope.Subject
	date := buf.Envelope.Date

	var bodyText string
	for _, section := range buf.BodySection {
		bodyText = string(section.Bytes)
		break
	}

	incoming := platforms.IncomingMessage{
		Platform:   "email",
		SessionID:  "email_" + from,
		ChatID:     from,
		SenderID:   from,
		Text:       fmt.Sprintf("Subject: %s\n\n%s", subject, bodyText),
		ReceivedAt: date,
	}
	if incoming.ReceivedAt.IsZero() {
		incoming.ReceivedAt = time.Now()
	}

	select {
	case p.out <- incoming:
	case <-ctx.Done():
	default:
		slog.Warn("imap channel full, dropping message", "from", from)
	}
	return nil
}
