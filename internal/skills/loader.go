package skills

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Skill es la metadata de un skill (sin body, cargada en Discover).
type Skill struct {
	Name            string
	Description     string
	Version         string
	Category        string
	Path            string
	Platforms       []string
	RequiredEnvVars []EnvVar
	RequiredFiles   []string
	LinkedFiles     []string
	Disabled        bool
}

// LoadedSkill incluye el contenido completo (cargado con Load).
type LoadedSkill struct {
	Skill
	Content     string
	LinkedData  map[string]string // path relativo -> contenido
	ReadyStatus string            // "AVAILABLE" | "SETUP_NEEDED" | "UNSUPPORTED"
	SetupNote   string
}

// LoaderOptions configura el SkillLoader.
type LoaderOptions struct {
	Dirs     []string
	Disabled []string
}

// Loader descubre y carga skills desde el filesystem.
// Fase 7.
type Loader struct {
	opts     LoaderOptions
	skills   []Skill
	disabledSet map[string]bool
	mu       sync.RWMutex
}

func NewLoader(opts LoaderOptions) *Loader {
	ds := make(map[string]bool)
	for _, d := range opts.Disabled {
		ds[d] = true
	}
	return &Loader{opts: opts, disabledSet: ds}
}

// Discover escanea los directorios configurados por archivos SKILL.md.
// Lee solo el frontmatter (primeros 4096 bytes).
// Fase 7.
func (l *Loader) Discover(ctx context.Context) error {
	var found []Skill

	for _, dir := range l.opts.Dirs {
		dir = expandHome(dir)
		if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()

			buf := make([]byte, 4096)
			n, _ := f.Read(buf)
			data := buf[:n]

			fm, _, err := ParseFrontmatter(data)
			if err != nil || fm.Name == "" {
				return nil
			}

			category := filepath.Base(filepath.Dir(filepath.Dir(path)))
			if fm.Category != "" {
				category = fm.Category
			}

			found = append(found, Skill{
				Name:            fm.Name,
				Description:     fm.Description,
				Version:         fm.Version,
				Category:        category,
				Path:            path,
				Platforms:       fm.Platforms,
				RequiredEnvVars: fm.RequiredEnvVars,
				RequiredFiles:   fm.RequiredFiles,
				LinkedFiles:     fm.LinkedFiles,
				Disabled:        l.disabledSet[fm.Name],
			})
			return nil
		}); err != nil {
			return err
		}
	}

	l.mu.Lock()
	l.skills = found
	l.mu.Unlock()
	return nil
}

// List retorna skills disponibles (filtradas por plataforma, disabled y prereqs).
func (l *Loader) List(_ context.Context) []Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var result []Skill
	for _, s := range l.skills {
		if s.Disabled {
			continue
		}
		if !platformMatches(s.Platforms) {
			continue
		}
		result = append(result, s)
	}
	return result
}

// Load lee el contenido completo de un skill por nombre.
// Fase 7.
func (l *Loader) Load(_ context.Context, name string) (*LoadedSkill, error) {
	l.mu.RLock()
	var target *Skill
	for i := range l.skills {
		if l.skills[i].Name == name {
			target = &l.skills[i]
			break
		}
	}
	l.mu.RUnlock()

	if target == nil {
		return nil, nil
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, err
	}
	_, body, _ := ParseFrontmatter(data)

	ls := &LoadedSkill{
		Skill:      *target,
		Content:    string(body),
		LinkedData: make(map[string]string),
		ReadyStatus: "AVAILABLE",
	}

	// Fase 7: cargar linked files, validar prereqs
	return ls, nil
}

// BuildSystemPromptBlock genera el bloque de skills para inyectar en el system prompt.
func (l *Loader) BuildSystemPromptBlock() string {
	l.mu.RLock()
	skills := l.skills
	l.mu.RUnlock()

	categories := make(map[string][]Skill)
	for _, s := range skills {
		if s.Disabled || !platformMatches(s.Platforms) {
			continue
		}
		categories[s.Category] = append(categories[s.Category], s)
	}

	if len(categories) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Skills disponibles\n<available_skills>\n")
	for cat, ss := range categories {
		sb.WriteString("  " + cat + ":\n")
		for _, s := range ss {
			sb.WriteString("    - " + s.Name + ": " + s.Description + "\n")
		}
	}
	sb.WriteString("</available_skills>\n\nCarga el skill relevante con skill_view antes de responder si aplica.")
	return sb.String()
}

func platformMatches(platforms []string) bool {
	if len(platforms) == 0 {
		return true
	}
	current := runtime.GOOS
	alias := map[string]string{"darwin": "macos", "windows": "windows", "linux": "linux"}
	for _, p := range platforms {
		if p == current || p == alias[current] {
			return true
		}
	}
	return false
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
