package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const copilotTokenURL = "https://api.github.com/copilot_internal/v2/token"

// ResolveGitHubToken retorna el GitHub token desde variables de entorno.
// Busca en orden: COPILOT_GITHUB_TOKEN, GH_TOKEN, GITHUB_TOKEN.
func ResolveGitHubToken() (string, error) {
	for _, env := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("copilot: no se encontro GitHub token en COPILOT_GITHUB_TOKEN, GH_TOKEN o GITHUB_TOKEN")
}

// FetchCopilotToken intercambia un GitHub token por un token de Copilot.
// El token de Copilot tiene vida corta (~30 min) y debe renovarse periodicamente.
func FetchCopilotToken(ctx context.Context, githubToken string) (*TokenSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilotTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("copilot token request: %w", err)
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Editor-Version", "hermes-go/1.0")
	req.Header.Set("Editor-Plugin-Version", "hermes-go/1.0")
	req.Header.Set("User-Agent", "hermes-go")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("copilot token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("copilot token: HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("copilot token parse: %w", err)
	}

	var expiresAt time.Time
	if result.ExpiresAt != "" {
		expiresAt, _ = time.Parse(time.RFC3339, result.ExpiresAt)
	}
	return &TokenSet{
		AccessToken: result.Token,
		ExpiresAt:   expiresAt,
		TokenType:   "Bearer",
	}, nil
}
