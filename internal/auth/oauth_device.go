package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DeviceFlowOptions configura el device code flow.
type DeviceFlowOptions struct {
	ProviderName   string
	DeviceEndpoint string // endpoint para obtener device code
	TokenEndpoint  string
	ClientID       string
	Scopes         []string
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
}

// StartDeviceFlow inicia el OAuth device code flow y bloquea hasta que el
// usuario autoriza en el navegador. Guarda el token en el Store global.
func StartDeviceFlow(ctx context.Context, opts DeviceFlowOptions) (*TokenSet, error) {
	// Paso 1: solicitar device code
	dc, err := requestDeviceCode(ctx, opts)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\nAbre el navegador y visita: %s\n", dc.VerificationURI)
	fmt.Printf("Ingresa el codigo: %s\n\n", dc.UserCode)

	// Paso 2: poll hasta que el usuario autorice
	interval := time.Duration(dc.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		tok, err := pollToken(ctx, opts, dc.DeviceCode)
		if err != nil {
			return nil, err
		}
		if tok != nil {
			return tok, nil
		}
	}
	return nil, fmt.Errorf("auth: device code flow timeout para %s", opts.ProviderName)
}

func requestDeviceCode(ctx context.Context, opts DeviceFlowOptions) (*deviceCodeResponse, error) {
	form := url.Values{
		"client_id": {opts.ClientID},
		"scope":     {strings.Join(opts.Scopes, " ")},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.DeviceEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code: HTTP %d: %s", resp.StatusCode, body)
	}

	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("device code parse: %w", err)
	}
	return &dc, nil
}

// pollToken retorna nil, nil cuando el usuario aun no autorizo (authorization_pending).
// Retorna un error real en cualquier otra falla.
func pollToken(ctx context.Context, opts DeviceFlowOptions, deviceCode string) (*TokenSet, error) {
	form := url.Values{
		"client_id":   {opts.ClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.TokenEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("poll token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("poll token parse: %w", err)
	}

	switch tr.Error {
	case "":
		// exito
	case "authorization_pending", "slow_down":
		return nil, nil
	default:
		return nil, fmt.Errorf("token error: %s", tr.Error)
	}

	expiresAt := time.Time{}
	if tr.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return &TokenSet{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tr.TokenType,
	}, nil
}

// RefreshToken intenta renovar un token usando el refresh_token.
func RefreshToken(ctx context.Context, tokenURL, clientID, refreshToken string) (*TokenSet, error) {
	form := url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("refresh token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("refresh token parse: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("refresh token error: %s", tr.Error)
	}

	expiresAt := time.Time{}
	if tr.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return &TokenSet{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tr.TokenType,
	}, nil
}
