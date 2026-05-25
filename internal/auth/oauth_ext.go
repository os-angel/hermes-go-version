package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ExternalOAuthOptions configura el flow OAuth con redirect en navegador.
type ExternalOAuthOptions struct {
	ProviderName  string
	AuthURL       string
	TokenEndpoint string
	ClientID      string
	Scopes        []string
	// LocalPort donde se escucha el redirect (default: 9876)
	LocalPort int
}

// StartExternalOAuth abre el navegador para autorizar y captura el callback
// en un servidor HTTP local. Retorna el TokenSet al completar el flow.
func StartExternalOAuth(ctx context.Context, opts ExternalOAuthOptions) (*TokenSet, error) {
	if opts.LocalPort == 0 {
		opts.LocalPort = 9876
	}
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", opts.LocalPort)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", opts.LocalPort),
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			fmt.Fprintf(w, "Error: %s. Puedes cerrar esta ventana.", errParam)
			errCh <- fmt.Errorf("oauth callback error: %s", errParam)
			return
		}
		if code == "" {
			fmt.Fprintf(w, "Error: no se recibio codigo. Puedes cerrar esta ventana.")
			errCh <- fmt.Errorf("oauth callback: sin codigo")
			return
		}
		fmt.Fprintf(w, "Autorizacion exitosa. Puedes cerrar esta ventana.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	// Construir URL de autorización
	authURL := buildAuthURL(opts, redirectURI)
	fmt.Printf("\nAbre el navegador para autorizar %s:\n%s\n\n", opts.ProviderName, authURL)
	_ = openBrowser(authURL)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case code := <-codeCh:
		return exchangeCode(ctx, opts.TokenEndpoint, opts.ClientID, code, redirectURI)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("oauth timeout para %s", opts.ProviderName)
	}
}

func buildAuthURL(opts ExternalOAuthOptions, redirectURI string) string {
	params := url.Values{
		"client_id":     {opts.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"scope":         {strings.Join(opts.Scopes, " ")},
	}
	return opts.AuthURL + "?" + params.Encode()
}

func exchangeCode(ctx context.Context, tokenURL, clientID, code, redirectURI string) (*TokenSet, error) {
	form := url.Values{
		"client_id":    {clientID},
		"code":         {code},
		"redirect_uri": {redirectURI},
		"grant_type":   {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code HTTP: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("exchange code parse: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("exchange code error: %s", tr.Error)
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

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
