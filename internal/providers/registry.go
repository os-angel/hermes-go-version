package providers

import (
	"fmt"
	"strings"
	"sync"
)

// Registry mantiene el catalogo de proveedores disponibles.
type Registry struct {
	mu       sync.RWMutex
	profiles map[string]*ProviderProfile // name -> profile
	aliases  map[string]string           // alias -> name canonico
}

var defaultRegistry = &Registry{
	profiles: make(map[string]*ProviderProfile),
	aliases:  make(map[string]string),
}

func init() {
	registerBuiltin(defaultRegistry)
}

// Default retorna el registro global de proveedores.
func Default() *Registry { return defaultRegistry }

// Register agrega un proveedor. Llamar desde init() de paquetes de plugin.
func (r *Registry) Register(p *ProviderProfile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[p.Name] = p
	for _, alias := range p.Aliases {
		r.aliases[strings.ToLower(alias)] = p.Name
	}
}

// Get retorna el perfil por nombre o alias. nil si no existe.
func (r *Registry) Get(name string) *ProviderProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(name)
	if canonical, ok := r.aliases[key]; ok {
		return r.profiles[canonical]
	}
	return r.profiles[key]
}

// MustGet retorna el perfil o hace panic si no existe.
func (r *Registry) MustGet(name string) *ProviderProfile {
	p := r.Get(name)
	if p == nil {
		panic(fmt.Sprintf("provider %q not found", name))
	}
	return p
}

// List retorna todos los perfiles registrados.
func (r *Registry) List() []*ProviderProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ProviderProfile, 0, len(r.profiles))
	for _, p := range r.profiles {
		out = append(out, p)
	}
	return out
}
