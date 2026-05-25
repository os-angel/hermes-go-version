package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// SessionCacheOptions configura el cache.
type SessionCacheOptions struct {
	MaxSize        int
	TTL            time.Duration
	EvictionPeriod time.Duration
	Store          SessionStore
}

// SessionCache es un LRU con TTL + persistencia en disco.
// Fase 4.
type SessionCache struct {
	cache *lru.Cache[string, *Session]
	opts  SessionCacheOptions
	mu    sync.Mutex
}

func NewSessionCache(opts SessionCacheOptions) (*SessionCache, error) {
	if opts.MaxSize <= 0 {
		opts.MaxSize = 128
	}
	if opts.TTL <= 0 {
		opts.TTL = time.Hour
	}
	if opts.EvictionPeriod <= 0 {
		opts.EvictionPeriod = 5 * time.Minute
	}

	c, err := lru.New[string, *Session](opts.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("create lru: %w", err)
	}
	return &SessionCache{cache: c, opts: opts}, nil
}

// Start lanza el loop de eviction (corre hasta ctx.Done()).
func (c *SessionCache) Start(ctx context.Context) {
	go c.evictionLoop(ctx)
}

// GetOrCreate retorna la sesion existente (cache -> disk -> nueva).
// Fase 4.
func (c *SessionCache) GetOrCreate(ctx context.Context, id, platform string) (*Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sess, ok := c.cache.Get(id); ok {
		sess.Touch()
		return sess, nil
	}

	// Intentar cargar de disco
	if c.opts.Store != nil {
		sess, err := c.opts.Store.Load(ctx, id)
		if err != nil {
			slog.Warn("session load from disk failed", "id", id, "err", err)
		} else if sess != nil {
			sess.Touch()
			c.cache.Add(id, sess)
			return sess, nil
		}
	}

	// Crear nueva
	sess := NewSession(id, platform)
	c.cache.Add(id, sess)
	return sess, nil
}

// Get retorna la sesion sin crear. nil si no existe en cache.
func (c *SessionCache) Get(id string) *Session {
	c.mu.Lock()
	defer c.mu.Unlock()
	sess, _ := c.cache.Get(id)
	return sess
}

// Persist guarda la sesion en disco inmediatamente.
func (c *SessionCache) Persist(ctx context.Context, s *Session) error {
	if c.opts.Store == nil {
		return nil
	}
	return c.opts.Store.Save(ctx, s)
}

// Delete elimina sesion de cache y disco.
func (c *SessionCache) Delete(ctx context.Context, id string) error {
	c.mu.Lock()
	c.cache.Remove(id)
	c.mu.Unlock()
	if c.opts.Store != nil {
		return c.opts.Store.Delete(ctx, id)
	}
	return nil
}

// Shutdown persiste todas las sesiones activas.
func (c *SessionCache) Shutdown(ctx context.Context) error {
	if c.opts.Store == nil {
		return nil
	}
	c.mu.Lock()
	keys := c.cache.Keys()
	c.mu.Unlock()
	for _, k := range keys {
		if sess, ok := c.cache.Get(k); ok {
			if err := c.opts.Store.Save(ctx, sess); err != nil {
				slog.Warn("session persist on shutdown failed", "id", k, "err", err)
			}
		}
	}
	return nil
}

func (c *SessionCache) evictionLoop(ctx context.Context) {
	ticker := time.NewTicker(c.opts.EvictionPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *SessionCache) evictExpired(ctx context.Context) {
	c.mu.Lock()
	keys := c.cache.Keys()
	c.mu.Unlock()

	now := time.Now()
	for _, k := range keys {
		sess, ok := c.cache.Get(k)
		if !ok {
			continue
		}
		if now.Sub(sess.LastUsed) > c.opts.TTL {
			if c.opts.Store != nil {
				if err := c.opts.Store.Save(ctx, sess); err != nil {
					slog.Warn("eviction persist failed", "id", k, "err", err)
				}
			}
			c.mu.Lock()
			c.cache.Remove(k)
			c.mu.Unlock()
		}
	}
}
