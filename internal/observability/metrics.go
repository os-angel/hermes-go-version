package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics agrupa los contadores y histogramas del agente.
type Metrics struct {
	MessagesProcessed *prometheus.CounterVec
	MessageLatency    *prometheus.HistogramVec
	ActiveSessions    prometheus.Gauge
	LLMCallErrors     *prometheus.CounterVec
	ToolCallDuration  *prometheus.HistogramVec
}

// NewMetrics registra y retorna las metricas Prometheus.
func NewMetrics() *Metrics {
	return &Metrics{
		MessagesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{Name: "agent_messages_total", Help: "Mensajes procesados por plataforma y estado."},
			[]string{"platform", "status"},
		),
		MessageLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_message_duration_seconds",
				Help:    "Latencia de procesamiento de mensajes.",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"platform"},
		),
		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{Name: "agent_active_sessions", Help: "Sesiones activas en cache."},
		),
		LLMCallErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{Name: "agent_llm_errors_total", Help: "Errores de llamadas al LLM."},
			[]string{"error_type"},
		),
		ToolCallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_tool_duration_seconds",
				Help:    "Duracion de ejecucion de tools.",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
			},
			[]string{"tool_name"},
		),
	}
}
