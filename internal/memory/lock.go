package memory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// FileLock es un wrapper sobre gofrs/flock para locking cross-platform.
type FileLock struct {
	fl *flock.Flock
}

// AcquireLock adquiere un lock exclusivo en <path>.lock.
// Llamar Release() cuando se termine.
func AcquireLock(path string) (*FileLock, error) {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}
	fl := flock.New(lockPath)
	if err := fl.Lock(); err != nil {
		return nil, fmt.Errorf("acquire lock %s: %w", lockPath, err)
	}
	return &FileLock{fl: fl}, nil
}

// Release libera el lock.
func (l *FileLock) Release() error {
	return l.fl.Unlock()
}
