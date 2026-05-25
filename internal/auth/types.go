package auth

import "time"

// TokenSet contiene las credenciales OAuth de un proveedor.
type TokenSet struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type,omitempty"`
}

// IsExpired reporta si el token expiro o expira en menos de 2 minutos.
func (t TokenSet) IsExpired() bool {
	return !t.ExpiresAt.IsZero() && time.Now().Add(2*time.Minute).After(t.ExpiresAt)
}

// Credential asocia un proveedor con su credencial almacenada.
type Credential struct {
	ProviderName string    `json:"provider"`
	Token        *TokenSet `json:"token,omitempty"`
	APIKey       string    `json:"api_key,omitempty"`
	SavedAt      time.Time `json:"saved_at"`
}
