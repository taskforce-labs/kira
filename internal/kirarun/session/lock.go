package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// RunLock holds an exclusive lock for a single run id.
type RunLock struct {
	f *flock.Flock
}

// TryLock acquires an exclusive lock for this run, or returns an error if busy.
func TryLock(lockPath string) (*RunLock, error) {
	if !filepath.IsAbs(lockPath) {
		return nil, fmt.Errorf("lock path must be absolute: %s", lockPath)
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), sessionDirMode); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	f := flock.New(lockPath)
	ok, err := f.TryLock()
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", lockPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("another process is already running this workflow (lock file: %s)", lockPath)
	}
	return &RunLock{f: f}, nil
}

// Unlock releases the lock.
func (l *RunLock) Unlock() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Unlock()
}
