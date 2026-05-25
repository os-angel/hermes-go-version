package mcp

import "os"

// FilterEnv construye un slice de "KEY=VALUE" para el subproceso del servidor MCP,
// incluyendo solo las variables de la whitelist. Si allowList esta vacio, hereda
// todas las variables del proceso padre.
func FilterEnv(allowList []string) []string {
	if len(allowList) == 0 {
		return os.Environ()
	}
	result := make([]string, 0, len(allowList))
	for _, key := range allowList {
		if val, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+val)
		}
	}
	return result
}
