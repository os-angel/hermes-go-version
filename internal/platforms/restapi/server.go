package restapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"hermes-go/internal/platforms"
)

// Server expone un endpoint REST para enviar mensajes al agente via HTTP.
// POST /v1/chat  {"session_id": "...", "message": "..."}
// Fase 12.
type Server struct {
	out chan<- platforms.IncomingMessage
	mux *chi.Mux
}

// chatRequest es el body del endpoint POST /v1/chat.
type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
}

// chatResponse es la respuesta inmediata (202 Accepted).
// La respuesta real se entrega por el canal de retorno configurado.
type chatResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

func NewServer(out chan<- platforms.IncomingMessage, tokens []string) *Server {
	s := &Server{out: out, mux: chi.NewRouter()}
	s.routes(tokens)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes(tokens []string) {
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Recoverer)
	if len(tokens) > 0 {
		s.mux.Use(BearerAuth(tokens))
	}
	s.mux.Post("/v1/chat", s.handleChat)
	s.mux.Get("/v1/health", s.handleHealth)
}

// handleChat acepta un mensaje y lo encola para el agente.
// Pendiente - Fase 12.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message required", http.StatusBadRequest)
		return
	}
	sessionID := req.SessionID
	if sessionID == "" {
		if req.UserID != "" {
			sessionID = "rest_" + req.UserID
		} else {
			sessionID = "rest_default"
		}
	}
	slog.Info("rest api chat", "session_id", sessionID)

	msg := platforms.IncomingMessage{
		Platform:   "rest",
		SessionID:  sessionID,
		ChatID:     sessionID,
		Text:       req.Message,
		ReceivedAt: time.Now(),
	}
	select {
	case s.out <- msg:
	case <-r.Context().Done():
		http.Error(w, "request cancelled", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(chatResponse{SessionID: sessionID, Status: "queued"})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) Shutdown(_ context.Context) error { return nil }
