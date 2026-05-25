package webhook

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"hermes-go/internal/platforms"
)

// Server escucha POST /webhooks/{id} y emite IncomingMessage.
// Valida la firma HMAC-SHA256 del payload antes de aceptar el mensaje.
// Fase 11.
type Server struct {
	store *SubscriptionStore
	out   chan<- platforms.IncomingMessage
	mux   *chi.Mux
}

func NewServer(store *SubscriptionStore, out chan<- platforms.IncomingMessage) *Server {
	s := &Server{store: store, out: out, mux: chi.NewRouter()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Post("/webhooks/{id}", s.handleWebhook)
}

// handleWebhook valida HMAC y emite el payload como IncomingMessage.
// Pendiente - Fase 11.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	slog.Info("webhook received", "id", id)
	http.Error(w, "not implemented: Phase 11", http.StatusNotImplemented)
}

// verifyHMAC compara la firma X-Hub-Signature-256 con el payload.
// Pendiente - Fase 11.
func verifyHMAC(secret string, payload []byte, signature string) bool {
	panic("not implemented: Phase 11")
}

// Shutdown cierra el servidor de forma ordenada.
func (s *Server) Shutdown(_ context.Context) error { return nil }
