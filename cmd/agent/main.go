package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"hermes-go/internal/agent"
	"hermes-go/internal/config"
	"hermes-go/internal/cron"
	"hermes-go/internal/llm"
	"hermes-go/internal/memory"
	"hermes-go/internal/observability"
	"hermes-go/internal/platforms"
	"hermes-go/internal/platforms/restapi"
	"hermes-go/internal/platforms/webhook"
	"hermes-go/internal/platforms/whatsapp"
	"hermes-go/internal/plugins"
	"hermes-go/internal/shutdown"
	"hermes-go/internal/skills"
	"hermes-go/internal/tools"

	// Importar paquetes de tools para que sus init() se ejecuten.
	_ "hermes-go/internal/tools/file"
)

func main() {
	// Subcomandos que no requieren config.yaml
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		runAuthCommand(os.Args[2:])
		return
	}

	cfgPath := flag.String("config", "config.yaml", "ruta al archivo de configuracion")
	flag.Parse()

	cfg := config.Must(config.Load(*cfgPath))

	logger := observability.NewLogger(cfg.Logging)
	slog.SetDefault(logger)

	if err := config.EnsureDirs(); err != nil {
		slog.Error("ensure dirs", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sd := shutdown.NewManager(30 * time.Second)

	// --- Tool Registry ---
	reg := tools.Default()

	// --- Plugin Registry ---
	pluginReg := plugins.Default()
	if err := pluginReg.InitAll(ctx, reg); err != nil {
		slog.Error("plugin init", "err", err)
		os.Exit(1)
	}
	sd.Register("plugins", shutdownFunc(func(ctx context.Context) error {
		return pluginReg.ShutdownAll(ctx)
	}))

	// --- Memory ---
	builtin := memory.NewBuiltinProvider(
		config.MemoriesDir(),
		cfg.Memory.Builtin.MemoryCharLimit,
		cfg.Memory.Builtin.UserCharLimit,
	)
	memMgr := memory.NewManager(builtin)
	memory.RegisterMemoryTool(reg, builtin)
	sd.Register("memory", memMgr)

	// --- LLM Client ---
	llmClient, err := llm.NewClient(ctx, cfg.LLM)
	if err != nil {
		slog.Error("llm client", "err", err)
		os.Exit(1)
	}

	// --- Skills Loader ---
	skillDirs := cfg.Skills.Dirs
	if len(skillDirs) == 0 {
		skillDirs = []string{config.SkillsDir()}
	}
	skillsLoader := skills.NewLoader(skills.LoaderOptions{
		Dirs:     skillDirs,
		Disabled: cfg.Skills.Disabled,
	})

	// --- Session Store + Cache ---
	store := agent.NewFileSessionStore(config.SessionsDir())
	cache, err := agent.NewSessionCache(agent.SessionCacheOptions{
		MaxSize: cfg.Agent.SessionCacheSize,
		TTL:     cfg.Agent.SessionTTL,
		Store:   store,
	})
	if err != nil {
		slog.Error("session cache", "err", err)
		os.Exit(1)
	}
	sd.Register("session_cache", cache)

	// --- Prompt Builder ---
	promptBuilder := agent.NewPromptBuilder(agent.PromptOptions{
		Identity: cfg.Agent.Identity,
		Memory:   memMgr,
		Skills:   skillsLoader,
		Model:    cfg.LLM.Model,
	})

	// --- Conversation Loop factory ---
	newLoop := func() *agent.ConversationLoop {
		return agent.NewConversationLoop(agent.LoopOptions{
			LLM:        llmClient,
			Registry:   reg,
			Memory:     memMgr,
			Skills:     skillsLoader,
			Prompt:     promptBuilder,
			MaxIter:    cfg.Agent.MaxIterations,
			ToolBudget: cfg.Agent.ToolBudgetChars,
		})
	}

	// --- Platform Router ---
	// router se declara primero para que la closure del handler pueda capturarlo por referencia.
	var router *platforms.Router
	loop := newLoop()
	router = platforms.NewRouter(512, func(ctx context.Context, msg platforms.IncomingMessage) error {
		sess, err := cache.GetOrCreate(ctx, msg.SessionID)
		if err != nil {
			return err
		}
		reply, err := loop.Run(ctx, sess, msg.Text)
		if err != nil {
			slog.Error("conversation loop", "session_id", msg.SessionID, "err", err)
			return err
		}
		return router.Send(ctx, platforms.OutgoingMessage{
			Platform: msg.Platform,
			ChatID:   msg.ChatID,
			Text:     reply,
			ReplyTo:  msg.MessageID,
		})
	})

	// --- WhatsApp ---
	if cfg.Platforms.WhatsApp.Enabled {
		waDir := config.WhatsAppDir()
		// Buscar bridge en orden: env var > ~/.hermes-go/bridge/bridge.js > ./bridge/bridge.js (dev)
		bridgePath := config.BridgeJSPath()
		if p := os.Getenv("HERMES_BRIDGE_JS"); p != "" {
			bridgePath = p
		} else if _, err := os.Stat(bridgePath); os.IsNotExist(err) {
			bridgePath = filepath.Join("bridge", "bridge.js")
		}
		bridge, err := whatsapp.NewBridge(whatsapp.BridgeOptions{
			Workdir:    waDir,
			BridgeJS:   bridgePath,
			NodePath:   cfg.Platforms.WhatsApp.BridgeNodePath,
			Port:       cfg.Platforms.WhatsApp.BridgePort,
			Mode:       cfg.Platforms.WhatsApp.Mode,
		})
		if err != nil {
			slog.Error("whatsapp bridge init", "err", err)
			os.Exit(1)
		}
		if err := bridge.Start(ctx); err != nil {
			slog.Error("whatsapp bridge start", "err", err)
			os.Exit(1)
		}
		sd.Register("whatsapp_bridge", bridge)

		waClient := whatsapp.NewClient(bridge.URL())
		waSender := whatsapp.NewSender(waClient)
		router.AddSender(waSender)

		waPoller := whatsapp.NewPoller(waClient, router.Incoming(), nil, time.Second)
		go func() {
			if err := waPoller.Start(ctx); err != nil {
				slog.Error("whatsapp poller", "err", err)
			}
		}()
	}

	// --- Skills tools ---
	// Registrar despues de que skillsLoader este construido.
	// (tools/skills/list.go y tools/skills/view.go se registran via RegisterXxx)

	// --- HTTP Server (webhook + rest api + metrics + health) ---
	httpMux := chi.NewRouter()
	httpMux.Use(middleware.RealIP)
	httpMux.Use(middleware.Recoverer)
	httpMux.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if cfg.Logging.PrometheusEnabled {
		httpMux.Handle("/metrics", promhttp.Handler())
	}

	if cfg.Platforms.Webhook.Enabled {
		subPath := cfg.Platforms.Webhook.SubscriptionsPath
		if subPath == "" {
			subPath = config.WebhookSubscriptionsPath()
		}
		subStore, err := webhook.NewSubscriptionStore(subPath)
		if err != nil {
			slog.Error("webhook subscription store", "err", err)
			os.Exit(1)
		}
		whServer := webhook.NewServer(subStore, router.Incoming())
		httpMux.Mount("/", whServer)
	}

	if cfg.Platforms.RESTAPI.Enabled {
		raServer := restapi.NewServer(router.Incoming(), cfg.Platforms.RESTAPI.Tokens)
		httpMux.Mount("/", raServer)
	}

	httpServer := &http.Server{
		Addr:         cfg.Server.ListenAddr,
		Handler:      httpMux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
	go func() {
		slog.Info("http server start", "addr", cfg.Server.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server", "err", err)
		}
	}()
	sd.Register("http_server", shutdownFunc(func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	}))

	// --- Cron Scheduler ---
	if cfg.Cron.Enabled {
		cronStore, err := cron.NewStore(config.CronJobsPath())
		if err != nil {
			slog.Error("cron store", "err", err)
			os.Exit(1)
		}
		cronRunner := cron.NewRunner(config.CronOutputDir(), func(ctx context.Context, job *cron.Job) error {
			msg := platforms.IncomingMessage{
				Platform:  job.Platform,
				SessionID: "cron_" + job.ID,
				ChatID:    job.ChatID,
				Text:      job.Prompt,
			}
			select {
			case router.Incoming() <- msg:
			case <-ctx.Done():
			}
			return nil
		})
		cronSched := cron.NewScheduler(cronStore, cronRunner)
		if err := cronSched.Start(ctx); err != nil {
			slog.Error("cron scheduler start", "err", err)
		}
		sd.Register("cron", cronSched)
	}

	// --- Start router workers ---
	router.Start(ctx, cfg.Agent.Workers)

	slog.Info("hermes-go started")
	sd.WaitForSignal()
}

// shutdownFunc adapta una funcion al tipo shutdown.Shutdowner.
type shutdownFunc func(ctx context.Context) error

func (f shutdownFunc) Shutdown(ctx context.Context) error { return f(ctx) }
