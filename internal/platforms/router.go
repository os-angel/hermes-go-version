package platforms

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// HandlerFunc procesa un mensaje entrante.
type HandlerFunc func(ctx context.Context, msg IncomingMessage) error

// Router distribuye mensajes entrantes a workers y enruta salidas al sender correcto.
// Fase 8.
type Router struct {
	incoming chan IncomingMessage
	senders  map[string]Sender
	handler  HandlerFunc
}

func NewRouter(bufferSize int, handler HandlerFunc) *Router {
	return &Router{
		incoming: make(chan IncomingMessage, bufferSize),
		senders:  make(map[string]Sender),
		handler:  handler,
	}
}

// AddSender registra un sender. Llamar antes de Start.
func (r *Router) AddSender(s Sender) {
	r.senders[s.Name()] = s
}

// Incoming retorna el canal para que los receivers pusheen mensajes.
func (r *Router) Incoming() chan<- IncomingMessage {
	return r.incoming
}

// Send enruta al sender correcto segun msg.Platform.
func (r *Router) Send(ctx context.Context, msg OutgoingMessage) error {
	s, ok := r.senders[msg.Platform]
	if !ok {
		return fmt.Errorf("no sender for platform %q", msg.Platform)
	}
	return s.Send(ctx, msg)
}

// Start lanza N workers. Bloquea hasta ctx.Done() o todos los workers terminan.
func (r *Router) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go r.worker(ctx)
	}
}

func (r *Router) worker(ctx context.Context) {
	for {
		select {
		case msg := <-r.incoming:
			if err := r.handler(ctx, msg); err != nil {
				slog.Error("message handler error",
					"platform", msg.Platform,
					"session_id", msg.SessionID,
					"err", err,
				)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Drain espera a que el canal se vacie con timeout.
func (r *Router) Drain(timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("drain timeout: %d messages remaining", len(r.incoming))
		default:
			if len(r.incoming) == 0 {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
