package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hermes-go/internal/config"
	"hermes-go/internal/providers"
)

// runSetupCommand ejecuta el wizard interactivo de configuracion.
// Equivalente a "hermes setup" en hermes-agent (Python).
func runSetupCommand(args []string) {
	reset := len(args) > 0 && args[0] == "--reset"

	cfgPath := filepath.Join(config.Home(), "config.yaml")
	envPath := filepath.Join(config.Home(), ".env")

	isFirstTime := !fileExists(cfgPath)

	fmt.Println()
	fmt.Println("hermes-go setup")
	fmt.Println(strings.Repeat("-", 40))

	if !isFirstTime && !reset {
		fmt.Printf("Configuracion existente: %s\n", cfgPath)
		fmt.Println("Continuando... (usa --reset para empezar desde cero)")
		fmt.Println()
	}

	if !isFirstTime && reset {
		backupPath := cfgPath + ".bak." + time.Now().Format("20060102_150405")
		_ = os.Rename(cfgPath, backupPath)
		fmt.Printf("Backup guardado en: %s\n\n", backupPath)
	}

	sc := bufio.NewScanner(os.Stdin)

	// --- Paso 1: Proveedor LLM ---
	fmt.Println("Paso 1 de 4: Proveedor LLM")
	fmt.Println()

	apiKeyProviders, oauthProviders := categorizeProviders()

	fmt.Println("Proveedores con API key:")
	for i, p := range apiKeyProviders {
		fmt.Printf("  %2d. %-20s %s\n", i+1, p.DisplayName, p.Description)
	}
	fmt.Println()
	fmt.Println("Proveedores con login (OAuth/suscripcion):")
	for i, p := range oauthProviders {
		fmt.Printf("  %2d. %-20s %s\n", len(apiKeyProviders)+i+1, p.DisplayName, p.Description)
	}
	fmt.Printf("  %2d. %-20s %s\n", len(apiKeyProviders)+len(oauthProviders)+1, "custom", "URL y modelo propios")
	fmt.Println()

	all := append(apiKeyProviders, oauthProviders...)
	selectedProvider := promptChoice(sc, "Elige proveedor (numero)", 1, len(all)+1)

	var providerName, baseURL, apiKey, model string

	if selectedProvider <= len(apiKeyProviders) {
		p := apiKeyProviders[selectedProvider-1]
		providerName = p.Name
		baseURL = p.BaseURL

		fmt.Printf("\nProveedor: %s\n", p.DisplayName)
		if p.Name == "ollama" || p.Name == "lmstudio" {
			apiKey = "local"
			fmt.Println("Proveedor local — no requiere API key.")
		} else {
			envVar := ""
			if len(p.EnvVars) > 0 {
				envVar = p.EnvVars[0]
			}
			existing := ""
			if envVar != "" {
				existing = os.Getenv(envVar)
			}
			hint := ""
			if existing != "" {
				hint = "Enter para usar la del entorno"
			}
			apiKey = promptString(sc, fmt.Sprintf("API key%s", suffix(hint)), existing)
		}

		if len(p.FallbackModels) > 0 {
			fmt.Println("\nModelos recomendados:")
			for i, m := range p.FallbackModels {
				fmt.Printf("  %d. %s\n", i+1, m)
			}
			defModel := p.FallbackModels[0]
			model = promptString(sc, fmt.Sprintf("Modelo (Enter = %s)", defModel), defModel)
		} else {
			model = promptString(sc, "Modelo", "")
		}

	} else if selectedProvider <= len(all) {
		p := oauthProviders[selectedProvider-len(apiKeyProviders)-1]
		providerName = p.Name
		baseURL = p.BaseURL
		fmt.Printf("\nProveedor: %s\n", p.DisplayName)
		fmt.Printf("Este proveedor usa OAuth. Ejecuta: hermes-go auth add %s\n", p.Name)
		fmt.Println("Continuando con el modelo por defecto...")
		if len(p.FallbackModels) > 0 {
			model = p.FallbackModels[0]
		}

	} else {
		// custom
		providerName = "custom"
		baseURL = promptString(sc, "Base URL (ej: http://localhost:8080/v1)", "")
		apiKey = promptString(sc, "API key (Enter si no requiere)", "")
		model = promptString(sc, "Modelo", "")
	}

	// --- Paso 2: Identidad del agente ---
	fmt.Println()
	fmt.Println("Paso 2 de 4: Identidad del agente")
	fmt.Println()
	defaultIdentity := "Eres un asistente personal util, conciso y honesto."
	fmt.Printf("Descripcion del agente (Enter = por defecto):\n  \"%s\"\n> ", defaultIdentity)
	sc.Scan()
	identity := strings.TrimSpace(sc.Text())
	if identity == "" {
		identity = defaultIdentity
	}

	// --- Paso 3: Plataformas ---
	fmt.Println()
	fmt.Println("Paso 3 de 4: Plataformas de mensajeria")
	fmt.Println()

	enableREST := promptYN(sc, "Habilitar REST API (acceso via curl/HTTP)", true)
	var restToken string
	if enableREST {
		restToken = generateToken()
		fmt.Printf("Token generado: %s\n", restToken)
	}

	enableWhatsApp := promptYN(sc, "Habilitar WhatsApp (requiere Node.js >= 18)", false)
	if enableWhatsApp {
		fmt.Println("  Al iniciar, escanea el QR con WhatsApp en tu telefono.")
	}

	enableEmail := promptYN(sc, "Habilitar Email (IMAP/SMTP)", false)
	var imapHost, imapUser, imapPass, smtpHost, smtpFrom string
	imapPort := 993
	smtpPort := 587
	if enableEmail {
		fmt.Println()
		imapHost = promptString(sc, "IMAP host (ej: imap.gmail.com)", "imap.gmail.com")
		imapPort = promptInt(sc, "IMAP port", 993)
		imapUser = promptString(sc, "Email / usuario IMAP", "")
		imapPass = promptString(sc, "Contrasena IMAP (o app password)", "")
		smtpHost = promptString(sc, "SMTP host (ej: smtp.gmail.com)", "smtp.gmail.com")
		smtpPort = promptInt(sc, "SMTP port", 587)
		smtpFrom = promptString(sc, "Direccion FROM", imapUser)
	}

	// --- Paso 4: Guardar ---
	fmt.Println()
	fmt.Println("Paso 4 de 4: Guardando configuracion...")
	fmt.Println()

	if err := os.MkdirAll(config.Home(), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "error: no se pudo crear %s: %v\n", config.Home(), err)
		os.Exit(1)
	}

	// Escribir .env
	envLines := buildEnv(providerName, apiKey, imapPass, restToken)
	if err := os.WriteFile(envPath, []byte(envLines), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "error escribiendo .env: %v\n", err)
		os.Exit(1)
	}

	// Escribir config.yaml
	yaml := buildConfig(configParams{
		providerName:   providerName,
		baseURL:        baseURL,
		model:          model,
		identity:       identity,
		enableREST:     enableREST,
		enableWhatsApp: enableWhatsApp,
		enableEmail:    enableEmail,
		imapHost:       imapHost,
		imapPort:       imapPort,
		imapUser:       imapUser,
		smtpHost:       smtpHost,
		smtpPort:       smtpPort,
		smtpFrom:       smtpFrom,
	})
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o640); err != nil {
		fmt.Fprintf(os.Stderr, "error escribiendo config.yaml: %v\n", err)
		os.Exit(1)
	}

	// --- Resumen ---
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Setup completado.")
	fmt.Println()
	fmt.Printf("  Configuracion:  %s\n", cfgPath)
	fmt.Printf("  Variables env:  %s\n", envPath)
	fmt.Println()
	fmt.Println("Plataformas habilitadas:")
	if enableREST {
		fmt.Printf("  REST API   -> POST http://localhost:8080/v1/chat  (token: %s)\n", restToken)
	}
	if enableWhatsApp {
		fmt.Println("  WhatsApp   -> escanea el QR al iniciar")
	}
	if enableEmail {
		fmt.Printf("  Email      -> IMAP %s / SMTP %s\n", imapHost, smtpHost)
	}
	fmt.Println()
	fmt.Println("Para iniciar el agente:")
	fmt.Println()
	fmt.Printf("  hermes-go --config %s\n", cfgPath)
	fmt.Println()
	if enableREST {
		fmt.Println("Para probar via REST:")
		fmt.Println()
		fmt.Printf("  curl -X POST http://localhost:8080/v1/chat \\\n")
		fmt.Printf("    -H \"Authorization: Bearer %s\" \\\n", restToken)
		fmt.Printf("    -H \"Content-Type: application/json\" \\\n")
		fmt.Printf("    -d '{\"message\": \"Hola\", \"session_id\": \"test\"}'\n")
		fmt.Println()
	}
}

// --- helpers ---

type configParams struct {
	providerName   string
	baseURL        string
	model          string
	identity       string
	enableREST     bool
	enableWhatsApp bool
	enableEmail    bool
	imapHost       string
	imapPort       int
	imapUser       string
	smtpHost       string
	smtpPort       int
	smtpFrom       string
}

func buildConfig(p configParams) string {
	var b strings.Builder

	envRef := func(key string) string { return fmt.Sprintf("${%s}", key) }

	b.WriteString("llm:\n")
	if p.providerName != "" && p.providerName != "custom" {
		b.WriteString(fmt.Sprintf("  provider: %q\n", p.providerName))
	}
	if p.baseURL != "" {
		b.WriteString(fmt.Sprintf("  base_url: %q\n", p.baseURL))
	}
	apiKeyEnv := providerEnvVar(p.providerName)
	if apiKeyEnv != "" {
		b.WriteString(fmt.Sprintf("  api_key: %q\n", envRef(apiKeyEnv)))
	}
	b.WriteString(fmt.Sprintf("  model: %q\n", p.model))
	b.WriteString("  timeout: 120s\n")
	b.WriteString("  max_retries: 5\n")
	b.WriteString("\n")

	b.WriteString("memory:\n")
	b.WriteString("  builtin:\n")
	b.WriteString("    memory_char_limit: 2200\n")
	b.WriteString("    user_char_limit: 1375\n")
	b.WriteString("\n")

	b.WriteString("skills:\n")
	b.WriteString(fmt.Sprintf("  dirs:\n    - %q\n", filepath.Join(config.Home(), "skills")))
	b.WriteString("\n")

	b.WriteString("platforms:\n")

	b.WriteString("  whatsapp:\n")
	b.WriteString(fmt.Sprintf("    enabled: %v\n", p.enableWhatsApp))
	if p.enableWhatsApp {
		b.WriteString("    backend: \"baileys\"\n")
		b.WriteString("    bridge_port: 3001\n")
		b.WriteString("    bridge_node_path: \"node\"\n")
		b.WriteString("    mode: \"bot\"\n")
	}

	b.WriteString("  email:\n")
	b.WriteString(fmt.Sprintf("    enabled: %v\n", p.enableEmail))
	if p.enableEmail {
		b.WriteString("    imap:\n")
		b.WriteString(fmt.Sprintf("      host: %q\n", p.imapHost))
		b.WriteString(fmt.Sprintf("      port: %d\n", p.imapPort))
		b.WriteString(fmt.Sprintf("      user: %q\n", p.imapUser))
		b.WriteString(fmt.Sprintf("      pass: %q\n", envRef("EMAIL_PASS")))
		b.WriteString("      mailbox: \"INBOX\"\n")
		b.WriteString("      poll_interval: 30s\n")
		b.WriteString("    smtp:\n")
		b.WriteString(fmt.Sprintf("      host: %q\n", p.smtpHost))
		b.WriteString(fmt.Sprintf("      port: %d\n", p.smtpPort))
		b.WriteString(fmt.Sprintf("      user: %q\n", p.imapUser))
		b.WriteString(fmt.Sprintf("      pass: %q\n", envRef("EMAIL_PASS")))
		b.WriteString(fmt.Sprintf("      from: %q\n", p.smtpFrom))
		b.WriteString("      starttls: true\n")
	}

	b.WriteString("  webhook:\n")
	b.WriteString("    enabled: true\n")
	b.WriteString(fmt.Sprintf("    subscriptions_path: %q\n", filepath.Join(config.Home(), "webhook_subscriptions.json")))

	b.WriteString("  restapi:\n")
	b.WriteString(fmt.Sprintf("    enabled: %v\n", p.enableREST))
	if p.enableREST {
		b.WriteString(fmt.Sprintf("    tokens:\n      - %q\n", envRef("HERMES_API_TOKEN")))
		b.WriteString("    require_token: true\n")
	}

	b.WriteString("\nmcp:\n  servers: {}\n")
	b.WriteString("\ncron:\n  enabled: true\n  tick_interval: 60s\n  grace_period: 10m\n")

	b.WriteString("\nagent:\n")
	b.WriteString(fmt.Sprintf("  identity: |\n    %s\n", strings.ReplaceAll(p.identity, "\n", "\n    ")))
	b.WriteString(fmt.Sprintf("  pii_salt: %q\n", envRef("HERMES_PII_SALT")))
	b.WriteString("  max_iterations: 12\n")
	b.WriteString("  tool_budget_chars: 60000\n")
	b.WriteString("  workers: 16\n")
	b.WriteString("  session_cache_size: 256\n")
	b.WriteString("  session_ttl: 1h\n")

	b.WriteString("\nserver:\n")
	b.WriteString("  listen_addr: \"0.0.0.0:8080\"\n")
	b.WriteString("  read_timeout: 30s\n")
	b.WriteString("  write_timeout: 60s\n")

	b.WriteString("\nlogging:\n")
	b.WriteString("  level: \"info\"\n")
	b.WriteString("  format: \"json\"\n")
	b.WriteString("  prometheus_enabled: false\n")

	return b.String()
}

func buildEnv(providerName, apiKey, emailPass, restToken string) string {
	var b strings.Builder
	b.WriteString("# hermes-go environment variables\n")
	b.WriteString("# Generado por hermes-go setup\n\n")

	envVar := providerEnvVar(providerName)
	if envVar != "" && apiKey != "" && apiKey != "local" {
		b.WriteString(fmt.Sprintf("%s=%s\n", envVar, apiKey))
	}
	if emailPass != "" {
		b.WriteString(fmt.Sprintf("EMAIL_PASS=%s\n", emailPass))
	}
	if restToken != "" {
		b.WriteString(fmt.Sprintf("HERMES_API_TOKEN=%s\n", restToken))
	}
	piiSalt := generateToken()[:16]
	b.WriteString(fmt.Sprintf("HERMES_PII_SALT=%s\n", piiSalt))

	return b.String()
}

func providerEnvVar(name string) string {
	reg := providers.Default()
	p := reg.Get(name)
	if p != nil && len(p.EnvVars) > 0 {
		return p.EnvVars[0]
	}
	if name == "custom" {
		return "LLM_API_KEY"
	}
	return ""
}

func categorizeProviders() (apiKey, oauth []*providers.ProviderProfile) {
	for _, p := range providers.Default().List() {
		switch p.AuthType {
		case providers.AuthTypeAPIKey, providers.AuthTypeAWSSDK:
			apiKey = append(apiKey, p)
		default:
			oauth = append(oauth, p)
		}
	}
	return
}

func promptString(sc *bufio.Scanner, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	sc.Scan()
	v := strings.TrimSpace(sc.Text())
	if v == "" {
		return defaultVal
	}
	return v
}

func promptYN(sc *bufio.Scanner, label string, defaultYes bool) bool {
	hint := "s/N"
	if defaultYes {
		hint = "S/n"
	}
	fmt.Printf("%s [%s]: ", label, hint)
	sc.Scan()
	v := strings.ToLower(strings.TrimSpace(sc.Text()))
	if v == "" {
		return defaultYes
	}
	return v == "s" || v == "si" || v == "y" || v == "yes"
}

func promptInt(sc *bufio.Scanner, label string, defaultVal int) int {
	fmt.Printf("%s [%d]: ", label, defaultVal)
	sc.Scan()
	v := strings.TrimSpace(sc.Text())
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func promptChoice(sc *bufio.Scanner, label string, min, max int) int {
	for {
		fmt.Printf("%s (%d-%d): ", label, min, max)
		sc.Scan()
		v := strings.TrimSpace(sc.Text())
		n, err := strconv.Atoi(v)
		if err == nil && n >= min && n <= max {
			return n
		}
		fmt.Printf("Ingresa un numero entre %d y %d.\n", min, max)
	}
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func suffix(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}
