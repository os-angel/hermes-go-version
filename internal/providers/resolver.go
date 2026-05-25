package providers

import (
	"context"
	"fmt"
	"os"

	"hermes-go/internal/auth"
)

// Resolved contiene todo lo necesario para construir un LLM client.
type Resolved struct {
	BaseURL        string
	APIKey         string
	DefaultHeaders map[string]string
	FallbackModels []string
}

// Resolve toma el nombre o alias de un proveedor y retorna las credenciales
// listas para usar. Intenta en orden:
//  1. Token OAuth almacenado en ~/.hermes-go/auth.json (si el proveedor usa OAuth)
//  2. Variables de entorno declaradas en EnvVars del perfil
//  3. Clave estatica "ollama" para proveedores locales sin autenticacion
func Resolve(ctx context.Context, providerName string) (*Resolved, error) {
	reg := Default()
	profile := reg.Get(providerName)
	if profile == nil {
		return nil, fmt.Errorf("proveedor %q no encontrado", providerName)
	}

	r := &Resolved{
		BaseURL:        profile.BaseURL,
		DefaultHeaders: profile.DefaultHeaders,
		FallbackModels: profile.FallbackModels,
	}

	switch profile.AuthType {
	case AuthTypeOAuthDevice, AuthTypeOAuthExternal:
		tok, err := resolveOAuth(ctx, profile)
		if err != nil {
			return nil, err
		}
		r.APIKey = tok.AccessToken

	case AuthTypeCopilot:
		tok, err := resolveCopilot(ctx, profile)
		if err != nil {
			return nil, err
		}
		r.APIKey = tok.AccessToken
		if r.DefaultHeaders == nil {
			r.DefaultHeaders = make(map[string]string)
		}
		r.DefaultHeaders["Editor-Version"] = "hermes-go/1.0"
		r.DefaultHeaders["Editor-Plugin-Version"] = "hermes-go/1.0"

	case AuthTypeAWSSDK:
		// AWS usa credenciales IAM via SDK, no API key
		r.APIKey = "aws-sdk"

	case AuthTypeAPIKey:
		key, err := resolveAPIKey(profile)
		if err != nil {
			return nil, err
		}
		r.APIKey = key
	}

	return r, nil
}

func resolveOAuth(ctx context.Context, profile *ProviderProfile) (*auth.TokenSet, error) {
	store := auth.Default()
	cred := store.Get(profile.Name)

	if cred != nil && cred.Token != nil && !cred.Token.IsExpired() {
		return cred.Token, nil
	}

	// Intentar renovar si hay refresh_token
	if cred != nil && cred.Token != nil && cred.Token.RefreshToken != "" && profile.OAuthTokenURL != "" {
		tok, err := auth.RefreshToken(ctx, profile.OAuthTokenURL, profile.OAuthClientID, cred.Token.RefreshToken)
		if err == nil {
			_ = store.Save(&auth.Credential{
				ProviderName: profile.Name,
				Token:        tok,
			})
			return tok, nil
		}
	}

	return nil, fmt.Errorf(
		"proveedor %q requiere autorizacion OAuth. Ejecuta: hermes-go auth add %s",
		profile.Name, profile.Name,
	)
}

func resolveCopilot(ctx context.Context, profile *ProviderProfile) (*auth.TokenSet, error) {
	store := auth.Default()
	cred := store.Get(profile.Name)

	// Token de Copilot valido
	if cred != nil && cred.Token != nil && !cred.Token.IsExpired() {
		return cred.Token, nil
	}

	// Necesitamos GitHub token para obtener uno nuevo
	ghToken, err := auth.ResolveGitHubToken()
	if err != nil {
		// fallback: si hay API key guardada usarla directamente
		if cred != nil && cred.APIKey != "" {
			return &auth.TokenSet{AccessToken: cred.APIKey, TokenType: "Bearer"}, nil
		}
		return nil, fmt.Errorf("copilot: %w", err)
	}

	tok, err := auth.FetchCopilotToken(ctx, ghToken)
	if err != nil {
		return nil, err
	}
	_ = store.Save(&auth.Credential{
		ProviderName: profile.Name,
		Token:        tok,
	})
	return tok, nil
}

func resolveAPIKey(profile *ProviderProfile) (string, error) {
	// Buscar en variables de entorno
	for _, envVar := range profile.EnvVars {
		if v := os.Getenv(envVar); v != "" {
			return v, nil
		}
	}

	// Proveedores locales (Ollama) no necesitan clave
	if len(profile.EnvVars) == 0 {
		return "ollama", nil
	}

	// Buscar credencial guardada manualmente
	cred := auth.Default().Get(profile.Name)
	if cred != nil && cred.APIKey != "" {
		return cred.APIKey, nil
	}

	vars := make([]string, len(profile.EnvVars))
	copy(vars, profile.EnvVars)
	return "", fmt.Errorf(
		"proveedor %q: no se encontro API key. Configura una de: %v",
		profile.Name, vars,
	)
}
