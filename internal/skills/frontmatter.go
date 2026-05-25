package skills

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// EnvVar describe una variable de entorno requerida por un skill.
type EnvVar struct {
	Name     string `yaml:"name"`
	Optional bool   `yaml:"optional"`
	Help     string `yaml:"help"`
}

// Frontmatter es el encabezado YAML de un SKILL.md.
type Frontmatter struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description"`
	Version         string            `yaml:"version"`
	Author          string            `yaml:"author"`
	License         string            `yaml:"license"`
	Platforms       []string          `yaml:"platforms"`
	Category        string            `yaml:"category"`
	RequiredEnvVars []EnvVar          `yaml:"required_environment_variables"`
	RequiredFiles   []string          `yaml:"required_credential_files"`
	LinkedFiles     []string          `yaml:"linked_files"`
	Metadata        map[string]any    `yaml:"metadata"`
}

var delimiter = []byte("---")

// ParseFrontmatter extrae el frontmatter YAML y el cuerpo markdown de un SKILL.md.
// Si no hay frontmatter, retorna Frontmatter vacio y el contenido completo como body.
func ParseFrontmatter(data []byte) (Frontmatter, []byte, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), delimiter) {
		return Frontmatter{}, data, nil
	}

	// Encontrar el segundo ---
	rest := bytes.TrimPrefix(bytes.TrimSpace(data), delimiter)
	rest = bytes.TrimLeft(rest, "\n\r")
	end := bytes.Index(rest, delimiter)
	if end == -1 {
		return Frontmatter{}, data, fmt.Errorf("unclosed frontmatter")
	}

	frontRaw := rest[:end]
	body := bytes.TrimLeft(rest[end+len(delimiter):], "\n\r")

	var fm Frontmatter
	if err := yaml.Unmarshal(frontRaw, &fm); err != nil {
		return Frontmatter{}, body, fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}
