package email

import (
	"mime"
	"strings"
	"time"
)

// ParsedEmail contiene los campos normalizados de un mensaje MIME.
// Fase 10.
type ParsedEmail struct {
	MessageID   string
	InReplyTo   string
	References  []string
	Subject     string
	From        Address
	To          []Address
	CC          []Address
	Date        time.Time
	TextBody    string
	HTMLBody    string
	Attachments []MailAttachment
	Headers     map[string]string
}

// Address representa un par nombre/email.
type Address struct {
	Name  string
	Email string
}

// MailAttachment es un adjunto decodificado del mensaje MIME.
type MailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// decodeHeader decodifica encoded-words RFC2047 en un header.
func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	out, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return out
}

// parseAddressList parsea "Name <email>, ..." en una lista de Address.
// Implementacion completa en Fase 10 con go-imap address structs.
func parseAddressList(raw string) []Address {
	if raw == "" {
		return nil
	}
	// fallback simple: split por coma
	var addrs []Address
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			addrs = append(addrs, Address{Email: part})
		}
	}
	return addrs
}

// Parse convierte los bytes raw de un mensaje MIME en ParsedEmail.
// Pendiente - Fase 10.
func Parse(raw []byte) (ParsedEmail, error) {
	panic("not implemented: Phase 10")
}
