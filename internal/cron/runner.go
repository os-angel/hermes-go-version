package cron

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Runner ejecuta un Job individual: escanea el prompt, construye el mensaje
// y lo envia al agente via el canal de entrada de la plataforma.
// Fase 15.
type Runner struct {
	outputDir string
	dispatch  func(ctx context.Context, job *Job) error
}

func NewRunner(outputDir string, dispatch func(ctx context.Context, job *Job) error) *Runner {
	return &Runner{outputDir: outputDir, dispatch: dispatch}
}

// Run ejecuta el job dado. Gracia de 60s: si el job deberia haber corrido
// hace mas de 60s (por restart del proceso), se omite para evitar doble ejecucion.
// Pendiente - Fase 15.
func (r *Runner) Run(ctx context.Context, job *Job) {
	if time.Since(job.NextRunAt) > 60*time.Second {
		slog.Info("cron job skipped (stale)", "job", job.ID, "next_run_at", job.NextRunAt)
		return
	}
	if err := ScanJobPrompt(job.Prompt); err != nil {
		slog.Error("cron job prompt rejected", "job", job.ID, "err", err)
		return
	}
	slog.Info("cron job start", "job", job.ID, "name", job.Name)
	panic("not implemented: Phase 15")
}

// saveOutput escribe el output del job en el directorio de outputs.
func (r *Runner) saveOutput(jobID string, output []byte) {
	if r.outputDir == "" {
		return
	}
	_ = os.MkdirAll(r.outputDir, 0o750)
	path := filepath.Join(r.outputDir, jobID+"_"+time.Now().Format("20060102T150405")+".txt")
	_ = os.WriteFile(path, output, 0o640)
}
