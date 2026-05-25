package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM       LLMConfig       `yaml:"llm"`
	Memory    MemoryConfig    `yaml:"memory"`
	Skills    SkillsConfig    `yaml:"skills"`
	Platforms PlatformsConfig `yaml:"platforms"`
	MCP       MCPConfig       `yaml:"mcp"`
	Cron      CronConfig      `yaml:"cron"`
	Agent     AgentConfig     `yaml:"agent"`
	Server    ServerConfig    `yaml:"server"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type LLMConfig struct {
	// Provider es el nombre del proveedor del registro (ej. "openrouter", "nous", "copilot").
	// Si se especifica, base_url y api_key se resuelven automaticamente.
	// Si no se especifica, se usan base_url y api_key directamente.
	Provider   string        `yaml:"provider"`
	BaseURL    string        `yaml:"base_url"`
	APIKey     string        `yaml:"api_key"`
	Model      string        `yaml:"model"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
}

type MemoryConfig struct {
	Builtin  BuiltinMemoryConfig `yaml:"builtin"`
	External string              `yaml:"external"`
}

type BuiltinMemoryConfig struct {
	MemoryCharLimit int `yaml:"memory_char_limit"`
	UserCharLimit   int `yaml:"user_char_limit"`
}

type SkillsConfig struct {
	Dirs     []string `yaml:"dirs"`
	Disabled []string `yaml:"disabled"`
}

type PlatformsConfig struct {
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`
	Email    EmailConfig    `yaml:"email"`
	Webhook  WebhookConfig  `yaml:"webhook"`
	RESTAPI  RESTAPIConfig  `yaml:"restapi"`
}

type WhatsAppConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Backend        string `yaml:"backend"`
	BridgePort     int    `yaml:"bridge_port"`
	BridgeNodePath string `yaml:"bridge_node_path"`
	Mode           string `yaml:"mode"`
	// Meta API fields (backend: meta)
	PhoneNumberID string `yaml:"phone_number_id"`
	AccessToken   string `yaml:"access_token"`
	VerifyToken   string `yaml:"verify_token"`
}

type EmailConfig struct {
	Enabled bool       `yaml:"enabled"`
	IMAP    IMAPConfig `yaml:"imap"`
	SMTP    SMTPConfig `yaml:"smtp"`
}

type IMAPConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	User         string        `yaml:"user"`
	Pass         string        `yaml:"pass"`
	Mailbox      string        `yaml:"mailbox"`
	UseIDLE      bool          `yaml:"use_idle"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

type SMTPConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	From      string `yaml:"from"`
	STARTTLS  bool   `yaml:"starttls"`
}

type WebhookConfig struct {
	Enabled           bool   `yaml:"enabled"`
	SubscriptionsPath string `yaml:"subscriptions_path"`
}

type RESTAPIConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Tokens       []string `yaml:"tokens"`
	RequireToken bool     `yaml:"require_token"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Name string `yaml:"-"` // poblado por MCPConfig.Servers key al cargar
	// stdio
	Command      string            `yaml:"command"`
	Args         []string          `yaml:"args"`
	Env          map[string]string `yaml:"env"`
	EnvAllowList []string          `yaml:"env_allow_list"` // vars a pasar al subproceso
	// http / sse
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Transport string          `yaml:"transport"` // "stdio" | "http" | "streamable_http" | "sse"
	// common
	Timeout                   time.Duration `yaml:"timeout"`
	ConnectTimeout            time.Duration `yaml:"connect_timeout"`
	SupportsParallelToolCalls bool          `yaml:"supports_parallel_tool_calls"`
}

type CronConfig struct {
	Enabled      bool          `yaml:"enabled"`
	TickInterval time.Duration `yaml:"tick_interval"`
	GracePeriod  time.Duration `yaml:"grace_period"`
}

type AgentConfig struct {
	Identity        string        `yaml:"identity"`
	PIISalt         string        `yaml:"pii_salt"`
	MaxIterations   int           `yaml:"max_iterations"`
	ToolBudgetChars int           `yaml:"tool_budget_chars"`
	Workers         int           `yaml:"workers"`
	SessionCacheSize int          `yaml:"session_cache_size"`
	SessionTTL      time.Duration `yaml:"session_ttl"`
}

type ServerConfig struct {
	ListenAddr   string        `yaml:"listen_addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type LoggingConfig struct {
	Level               string `yaml:"level"`
	Format              string `yaml:"format"`
	PrometheusEnabled   bool   `yaml:"prometheus_enabled"`
}

// Load carga config.yaml con expansion de env vars.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	data = []byte(os.ExpandEnv(string(data)))

	cfg := defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Must carga config o hace panic. Para uso en main.
func Must(cfg *Config, err error) *Config {
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

func defaults() *Config {
	return &Config{
		LLM: LLMConfig{
			Timeout:    120 * time.Second,
			MaxRetries: 5,
		},
		Memory: MemoryConfig{
			Builtin: BuiltinMemoryConfig{
				MemoryCharLimit: 2200,
				UserCharLimit:   1375,
			},
		},
		Cron: CronConfig{
			TickInterval: 60 * time.Second,
			GracePeriod:  10 * time.Minute,
		},
		Agent: AgentConfig{
			MaxIterations:    12,
			ToolBudgetChars:  60000,
			Workers:          16,
			SessionCacheSize: 256,
			SessionTTL:       time.Hour,
		},
		Server: ServerConfig{
			ListenAddr:   "0.0.0.0:8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
