package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Bridge gestiona el ciclo de vida del subproceso bridge.js (Node.js + Baileys).
// Fase 9.
type Bridge struct {
	workdir    string
	nodePath   string
	bridgeJS   string
	port       int
	mode       string
	pidFile    string
	cmd        *exec.Cmd
	baseURL    string
}

// BridgeOptions configura el Bridge.
type BridgeOptions struct {
	Workdir    string
	BridgeJS   string
	NodePath   string
	Port       int
	Mode       string
}

func NewBridge(opts BridgeOptions) (*Bridge, error) {
	nodePath := opts.NodePath
	if nodePath == "" {
		nodePath = "node"
	}
	port := opts.Port
	if port == 0 {
		port = 3001
	}
	mode := opts.Mode
	if mode == "" {
		mode = "bot"
	}
	if err := os.MkdirAll(opts.Workdir, 0o750); err != nil {
		return nil, fmt.Errorf("create workdir: %w", err)
	}
	return &Bridge{
		workdir:  opts.Workdir,
		nodePath: nodePath,
		bridgeJS: opts.BridgeJS,
		port:     port,
		mode:     mode,
		pidFile:  filepath.Join(opts.Workdir, "bridge.pid"),
		baseURL:  fmt.Sprintf("http://127.0.0.1:%d", port),
	}, nil
}

// URL retorna la base URL del bridge HTTP.
func (b *Bridge) URL() string { return b.baseURL }

// Start mata procesos huerfanos, lanza bridge.js y espera al health check.
// Fase 9.
func (b *Bridge) Start(ctx context.Context) error {
	b.killStale()

	args := []string{
		b.bridgeJS,
		"--port", strconv.Itoa(b.port),
		"--session", b.workdir,
		"--mode", b.mode,
	}
	b.cmd = exec.CommandContext(ctx, b.nodePath, args...)
	b.cmd.Dir = filepath.Dir(b.bridgeJS)

	// stderr al log file
	logPath := filepath.Join(b.workdir, "..", "logs", "bridge-stderr.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o750)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err == nil {
		b.cmd.Stderr = logFile
	}

	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("start bridge: %w", err)
	}

	// Guardar PID
	_ = os.WriteFile(b.pidFile, []byte(strconv.Itoa(b.cmd.Process.Pid)), 0o640)
	slog.Info("bridge started", "pid", b.cmd.Process.Pid, "port", b.port)

	// Esperar health check
	deadline := time.Now().Add(30 * time.Second)
	c := NewClient(b.baseURL)
	for time.Now().Before(deadline) {
		if err := c.Health(ctx); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("bridge did not become healthy within 30s")
}

// Stop envia SIGTERM y espera 5s; luego SIGKILL.
func (b *Bridge) Stop() error {
	if b.cmd == nil || b.cmd.Process == nil {
		return nil
	}
	_ = b.cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- b.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = b.cmd.Process.Kill()
	}
	_ = os.Remove(b.pidFile)
	return nil
}

// Shutdown implementa la interfaz de shutdown.
func (b *Bridge) Shutdown(_ context.Context) error { return b.Stop() }

// killStale mata el proceso anterior si el PID file existe.
func (b *Bridge) killStale() {
	data, err := os.ReadFile(b.pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	if runtime.GOOS == "windows" {
		exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
	} else {
		_ = proc.Signal(os.Interrupt)
		time.Sleep(500 * time.Millisecond)
		_ = proc.Kill()
	}
	_ = os.Remove(b.pidFile)
	slog.Info("killed stale bridge", "pid", pid)
}
