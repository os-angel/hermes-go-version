package plugins

import (
	"context"

	"hermes-go/internal/tools"
)

// Plugin es la interfaz que cada plugin en-arbol debe implementar.
// Los plugins se registran via init() en su propio paquete y se incluyen
// en el binario con un blank import en cmd/agent/main.go.
// Fase 16.
type Plugin interface {
	// Name retorna el identificador unico del plugin.
	Name() string

	// Init inicializa el plugin y registra sus tools en el registry.
	// Se llama una sola vez al arrancar el agente.
	Init(ctx context.Context, reg *tools.Registry) error

	// Shutdown libera recursos del plugin.
	Shutdown(ctx context.Context) error
}

// Metadata contiene informacion descriptiva del plugin.
type Metadata struct {
	Name        string
	Version     string
	Description string
	Author      string
}
