package providers

// AuthType define como se autentica un proveedor LLM.
type AuthType string

const (
	// AuthTypeAPIKey — clave de API via variable de entorno o config.
	AuthTypeAPIKey AuthType = "api_key"
	// AuthTypeOAuthDevice — OAuth device code flow (Nous Portal).
	// El usuario corre "hermes-go auth add <provider>", se abre el navegador
	// y el token se guarda en ~/.hermes-go/auth.json con refresh automatico.
	AuthTypeOAuthDevice AuthType = "oauth_device_code"
	// AuthTypeOAuthExternal — OAuth con redirect en navegador (Codex, xAI, MiniMax).
	AuthTypeOAuthExternal AuthType = "oauth_external"
	// AuthTypeCopilot — GitHub Copilot token especial.
	AuthTypeCopilot AuthType = "copilot"
	// AuthTypeAWSSDK — AWS Bedrock via credenciales IAM, sin clave explicita.
	AuthTypeAWSSDK AuthType = "aws_sdk"
)

// ProviderProfile describe un proveedor LLM: como autenticarse y donde conectarse.
// Equivalente a ProviderProfile en hermes-agent (providers/base.py).
type ProviderProfile struct {
	Name        string
	DisplayName string
	Description string
	SignupURL   string
	Aliases     []string

	// Auth
	AuthType AuthType
	EnvVars  []string // variables de entorno a buscar para la API key (en orden)

	// Endpoints
	BaseURL   string
	ModelsURL string // endpoint para listar modelos disponibles

	// Modelos
	FallbackModels  []string
	DefaultAuxModel string // modelo barato para tareas auxiliares

	// Headers adicionales para todas las requests
	DefaultHeaders map[string]string

	// OAuth (solo para AuthTypeOAuthDevice y AuthTypeOAuthExternal)
	OAuthPortalURL  string // URL donde el usuario autoriza
	OAuthTokenURL   string // endpoint para intercambiar code por token
	OAuthClientID   string
	OAuthScopes     []string
}
