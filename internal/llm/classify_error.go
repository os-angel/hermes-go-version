package llm

import (
	"errors"
	"net"
	"strings"

	"github.com/openai/openai-go"
)

type ErrorClass int

const (
	Transient  ErrorClass = iota // 429, 5xx, timeout, connection reset -> retry
	AuthError                    // 401, 403 -> no retry
	BadRequest                   // 400, 422 -> no retry
	Fatal                        // otros -> no retry
)

// Classify clasifica un error de la API para decidir si hacer retry.
func Classify(err error) ErrorClass {
	if err == nil {
		return Fatal
	}

	// Errores de red son transitorios
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return Transient
		}
	}

	// Connection reset / EOF
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "no such host") {
		return Transient
	}

	// openai-go expone APIError con StatusCode
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401, 403:
			return AuthError
		case 400, 422:
			return BadRequest
		case 429, 500, 502, 503, 504:
			return Transient
		default:
			if apiErr.StatusCode >= 500 {
				return Transient
			}
			return Fatal
		}
	}

	return Transient
}
