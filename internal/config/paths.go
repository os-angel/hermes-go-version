package config

import (
	"os"
	"path/filepath"
)

// Home retorna el directorio raiz de datos (~/.hermes-go o HERMES_GO_HOME).
func Home() string {
	if h := os.Getenv("HERMES_GO_HOME"); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hermes-go"
	}
	return filepath.Join(home, ".hermes-go")
}

func MemoriesDir() string    { return filepath.Join(Home(), "memories") }
func SessionsDir() string    { return filepath.Join(Home(), "sessions") }
func SkillsDir() string      { return filepath.Join(Home(), "skills") }
func CronDir() string        { return filepath.Join(Home(), "cron") }
func CronOutputDir() string  { return filepath.Join(Home(), "cron", "output") }
func WhatsAppDir() string    { return filepath.Join(Home(), "whatsapp") }
func BridgeDir() string      { return filepath.Join(Home(), "bridge") }
func BridgeJSPath() string   { return filepath.Join(BridgeDir(), "bridge.js") }
func LogsDir() string        { return filepath.Join(Home(), "logs") }

func CronJobsPath() string              { return filepath.Join(CronDir(), "jobs.json") }
func CronLockPath() string              { return filepath.Join(CronDir(), ".tick.lock") }
func WebhookSubscriptionsPath() string  { return filepath.Join(Home(), "webhook_subscriptions.json") }

// EnsureDirs crea los directorios base si no existen.
func EnsureDirs() error {
	dirs := []string{
		MemoriesDir(),
		SessionsDir(),
		SkillsDir(),
		CronDir(),
		CronOutputDir(),
		WhatsAppDir(),
		LogsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return err
		}
	}
	return nil
}
