package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
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
func parseAddressList(raw string) []Address {
	if raw == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(raw)
	if err != nil {
		// fallback: split por coma
		var out []Address
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, Address{Email: part})
			}
		}
		return out
	}
	out := make([]Address, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, Address{Name: a.Name, Email: a.Address})
	}
	return out
}

// parseReferences extrae los message IDs del header References.
func parseReferences(raw string) []string {
	if raw == "" {
		return nil
	}
	var refs []string
	for _, r := range strings.Fields(raw) {
		r = strings.Trim(r, "<>")
		if r != "" {
			refs = append(refs, r)
		}
	}
	return refs
}

// extractHeaders copia los headers relevantes al mapa de salida.
func extractHeaders(h mail.Header) map[string]string {
	out := make(map[string]string)
	for k := range h {
		out[k] = h.Get(k)
	}
	return out
}

// Parse convierte los bytes raw de un mensaje MIME en ParsedEmail.
func Parse(raw []byte) (ParsedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return ParsedEmail{}, fmt.Errorf("email parse: %w", err)
	}

	h := msg.Header
	fromList := parseAddressList(h.Get("From"))
	var from Address
	if len(fromList) > 0 {
		from = fromList[0]
	}

	date, _ := mail.ParseDate(h.Get("Date"))

	parsed := ParsedEmail{
		MessageID:  strings.Trim(h.Get("Message-Id"), "<>"),
		InReplyTo:  strings.Trim(h.Get("In-Reply-To"), "<>"),
		References: parseReferences(h.Get("References")),
		Subject:    decodeHeader(h.Get("Subject")),
		From:       from,
		To:         parseAddressList(h.Get("To")),
		CC:         parseAddressList(h.Get("Cc")),
		Date:       date,
		Headers:    extractHeaders(h),
	}

	contentType := h.Get("Content-Type")
	if strings.Contains(contentType, "multipart/") {
		parseMultipart(msg.Body, contentType, &parsed)
	} else {
		body, _ := io.ReadAll(msg.Body)
		if strings.Contains(contentType, "text/html") {
			parsed.HTMLBody = string(body)
		} else {
			parsed.TextBody = string(body)
		}
	}

	return parsed, nil
}

// parseMultipart extrae partes de un mensaje multipart MIME.
func parseMultipart(body io.Reader, contentType string, parsed *ParsedEmail) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return
	}
	boundary, ok := params["boundary"]
	if !ok {
		return
	}
	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		partCT := part.Header.Get("Content-Type")
		data, err := io.ReadAll(part)
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(partCT, "text/plain"):
			parsed.TextBody += string(data)
		case strings.Contains(partCT, "text/html"):
			parsed.HTMLBody += string(data)
		case strings.Contains(partCT, "multipart/"):
			parseMultipart(bytes.NewReader(data), partCT, parsed)
		default:
			filename := part.FileName()
			if filename != "" {
				parsed.Attachments = append(parsed.Attachments, MailAttachment{
					Filename:    filename,
					ContentType: partCT,
					Data:        data,
				})
			}
		}
	}
}
