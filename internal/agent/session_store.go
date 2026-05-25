package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SessionStore persiste sesiones en disco.
type SessionStore interface {
	Load(ctx context.Context, id string) (*Session, error)
	Save(ctx context.Context, s *Session) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]string, error)
}

// FileSessionStore guarda cada sesion en <dir>/<id>.json con atomic write.
// Fase 4.
type FileSessionStore struct {
	dir string
}

func NewFileSessionStore(dir string) SessionStore {
	return &FileSessionStore{dir: dir}
}

func (s *FileSessionStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *FileSessionStore) Load(_ context.Context, id string) (*Session, error) {
	data, err := os.ReadFile(s.path(id))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read session %s: %w", id, err)
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parse session %s: %w", id, err)
	}
	return &sess, nil
}

func (s *FileSessionStore) Save(_ context.Context, sess *Session) error {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	tmp := s.path(sess.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	return os.Rename(tmp, s.path(sess.ID))
}

func (s *FileSessionStore) Delete(_ context.Context, id string) error {
	err := os.Remove(s.path(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *FileSessionStore) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return ids, nil
}
