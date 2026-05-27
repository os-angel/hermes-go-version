package platforms

import (
	"context"
	"time"
)

// IncomingMessage es un mensaje normalizado de cualquier plataforma.
type IncomingMessage struct {
	Platform    string
	SessionID   string   // sha256("<platform>:<chat_id>")
	ChatID      string   // raw, para responder
	SenderID    string   // raw
	SenderName  string
	IsGroup     bool
	Text        string
	MessageID   string
	ReplyTo     string
	Attachments []Attachment
	Metadata    map[string]string
	ReceivedAt  time.Time
	// ReplyC es opcional. Si no es nil, el worker envia la respuesta aqui
	// en lugar de (o ademas de) usar el Sender de la plataforma.
	// Usado por la REST API para respuesta sincrona.
	ReplyC chan<- string
}

// Attachment es un archivo adjunto ya descargado localmente.
type Attachment struct {
	LocalPath string
	MimeType  string
	Filename  string
	SizeBytes int64
}

// OutgoingMessage es lo que el agente envia de vuelta.
type OutgoingMessage struct {
	Platform  string
	ChatID    string
	Text      string
	ReplyTo   string
	Media     []Attachment
	Typing    bool
	Metadata  map[string]string
}

// Sender envia mensajes a una plataforma especifica.
type Sender interface {
	Name() string
	Send(ctx context.Context, msg OutgoingMessage) error
	SendTyping(ctx context.Context, chatID string) error
}

// Receiver recibe mensajes y los empuja al canal del router.
type Receiver interface {
	Name() string
	Start(ctx context.Context, out chan<- IncomingMessage) error
	Stop() error
}
