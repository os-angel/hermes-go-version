package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sub, ok := s.store.Get(id)
	if !ok || !sub.Enabled {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if sub.Secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyHMAC(sub.Secret, body, sig) {
			slog.Warn("webhook hmac mismatch", "id", id)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}
	msg := platforms.IncomingMessage{
		Platform:   "webhook",
		SessionID:  "webhook_" + id,
		ChatID:     id,
		Text:       string(body),
		ReceivedAt: time.Now(),
	}
	select {
	case s.out <- msg:
	case <-r.Context().Done():
		http.Error(w, "timeout", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	slog.Info("webhook accepted", "id", id)
}

// verifyHMAC compara la firma X-Hub-Signature-256 con el payload usando constant-time compare.
func verifyHMAC(secret string, payload []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	hexSig := strings.TrimPrefix(signature, prefix)
	expected, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	computed := mac.Sum(nil)
	return subtle.ConstantTimeCompare(computed, expected) == 1
}

// Shutdown cierra el servidor de forma ordenada.
func (s *Server) Shutdown(_ context.Context) error { return nil }
