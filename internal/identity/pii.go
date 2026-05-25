package identity

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

var salt string

// SetSalt configura el salt para el hashing. Llamar en main antes de cualquier Hash().
func SetSalt(s string) { salt = s }

// Hash retorna sha256("v1:<salt>:<value>") en hex.
func Hash(value string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("v1:%s:%s", salt, value)))
	return fmt.Sprintf("%x", h)
}

// SessionID construye el id canonico de sesion: "<platform>_<hash(chatID)>".
func SessionID(platform, chatID string) string {
	return fmt.Sprintf("%s_%s", strings.ToLower(platform), Hash(chatID))
}
