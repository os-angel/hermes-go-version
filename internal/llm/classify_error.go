package llm

import (
	"errors"
	"net"
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
		return Transient
	}

	// TODO Fase 3: parsear codigos HTTP del openai-go error type
	return Transient
}
