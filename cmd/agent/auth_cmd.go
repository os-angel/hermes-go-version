package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"hermes-go/internal/auth"
	"hermes-go/internal/providers"
)

// runAuthCommand maneja el subcomando "hermes-go auth <accion> [proveedor]".
// Uso:
//
//	hermes-go auth list              — listar proveedores y estado
//	hermes-go auth add <proveedor>   — autorizar un proveedor
//	hermes-go auth remove <proveedor>— eliminar credencial
func runAuthCommand(args []string) {
	if len(args) == 0 {
		printAuthHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		authList()
	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "uso: hermes-go auth add <proveedor>")
			os.Exit(1)
		}
		authAdd(args[1])
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "uso: hermes-go auth remove <proveedor>")
			os.Exit(1)
		}
		authRemove(args[1])
	default:
		fmt.Fprintf(os.Stderr, "accion desconocida: %q\n", args[0])
		printAuthHelp()
		os.Exit(1)
	}
}

func printAuthHelp() {
	fmt.Println("Uso:")
	fmt.Println("  hermes-go auth list")
	fmt.Println("  hermes-go auth add <proveedor>")
	fmt.Println("  hermes-go auth remove <proveedor>")
}

func authList() {
	reg := providers.Default()
	all := reg.List()
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })

	store := auth.Default()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVEEDOR\tTIPO AUTH\tESTADO")
	fmt.Fprintln(w, "---------\t----------\t------")

	for _, p := range all {
		status := resolveStatus(store, p)
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.AuthType, status)
	}
	w.Flush()
}

func resolveStatus(store *auth.Store, p *providers.ProviderProfile) string {
	cred := store.Get(p.Name)

	switch p.AuthType {
	case providers.AuthTypeAPIKey:
		for _, env := range p.EnvVars {
			if os.Getenv(env) != "" {
				return "configurado (env: " + env + ")"
			}
		}
		if cred != nil && cred.APIKey != "" {
			return "configurado (guardado)"
		}
		if len(p.EnvVars) == 0 {
			return "listo (sin clave)"
		}
		return "sin configurar"

	case providers.AuthTypeOAuthDevice, providers.AuthTypeOAuthExternal:
		if cred != nil && cred.Token != nil {
			if cred.Token.IsExpired() {
				if cred.Token.RefreshToken != "" {
					return "token vencido (se renovara automaticamente)"
				}
				return "token vencido (requiere re-autorizacion)"
			}
			left := time.Until(cred.Token.ExpiresAt).Truncate(time.Minute)
			if cred.Token.ExpiresAt.IsZero() {
				return "autorizado"
			}
			return fmt.Sprintf("autorizado (expira en %s)", left)
		}
		return "sin autorizar — ejecuta: hermes-go auth add " + p.Name

	case providers.AuthTypeCopilot:
		for _, env := range []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"} {
			if os.Getenv(env) != "" {
				return "configurado (env: " + env + ")"
			}
		}
		return "sin configurar (requiere COPILOT_GITHUB_TOKEN, GH_TOKEN o GITHUB_TOKEN)"

	case providers.AuthTypeAWSSDK:
		return "usa credenciales IAM (AWS_PROFILE / env)"
	}
	return "desconocido"
}

func authAdd(providerName string) {
	reg := providers.Default()
	p := reg.Get(providerName)
	if p == nil {
		fmt.Fprintf(os.Stderr, "proveedor %q no encontrado\n", providerName)
		fmt.Fprintln(os.Stderr, "ejecuta 'hermes-go auth list' para ver los disponibles")
		os.Exit(1)
	}

	ctx := context.Background()
	store := auth.Default()

	switch p.AuthType {
	case providers.AuthTypeOAuthDevice:
		fmt.Printf("Autorizando %s via device code flow...\n", p.DisplayName)
		tok, err := auth.StartDeviceFlow(ctx, auth.DeviceFlowOptions{
			ProviderName:   p.Name,
			DeviceEndpoint: p.OAuthPortalURL, // almacena el device code endpoint directamente
			TokenEndpoint:  p.OAuthTokenURL,
			ClientID:       p.OAuthClientID,
			Scopes:         p.OAuthScopes,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(&auth.Credential{
			ProviderName: p.Name,
			Token:        tok,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "guardar credencial: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Autorizado correctamente. Token guardado.\n")

	case providers.AuthTypeOAuthExternal:
		fmt.Printf("Autorizando %s via OAuth en navegador...\n", p.DisplayName)
		tok, err := auth.StartExternalOAuth(ctx, auth.ExternalOAuthOptions{
			ProviderName:  p.Name,
			AuthURL:       p.OAuthPortalURL,
			TokenEndpoint: p.OAuthTokenURL,
			ClientID:      p.OAuthClientID,
			Scopes:        p.OAuthScopes,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := store.Save(&auth.Credential{
			ProviderName: p.Name,
			Token:        tok,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "guardar credencial: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Autorizado correctamente. Token guardado.\n")

	case providers.AuthTypeAPIKey:
		fmt.Fprintf(os.Stderr,
			"proveedor %q usa API key. Configura la variable de entorno %v o agrega api_key al config.\n",
			p.Name, p.EnvVars)

	case providers.AuthTypeCopilot:
		fmt.Fprintln(os.Stderr,
			"GitHub Copilot usa tu GitHub token. Configura COPILOT_GITHUB_TOKEN, GH_TOKEN o GITHUB_TOKEN.")

	case providers.AuthTypeAWSSDK:
		fmt.Fprintln(os.Stderr,
			"AWS Bedrock usa credenciales IAM. Configura AWS_PROFILE o AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY.")
	}
}

func authRemove(providerName string) {
	store := auth.Default()
	if err := store.Delete(providerName); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Credencial de %q eliminada.\n", providerName)
}
