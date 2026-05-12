package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FilesystemBackend stores content on the local filesystem.
type FilesystemBackend struct {
	basePath string
}

// NewFilesystemBackend creates a new filesystem storage backend.
func NewFilesystemBackend(basePath string) (*FilesystemBackend, error) {
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &FilesystemBackend{basePath: basePath}, nil
}

func (f *FilesystemBackend) Store(_ context.Context, key string, content io.Reader, _ int64) error {
	path := f.keyToPath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		os.Remove(path)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (f *FilesystemBackend) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path := f.keyToPath(key)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("content not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

func (f *FilesystemBackend) Delete(_ context.Context, key string) error {
	path := f.keyToPath(key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (f *FilesystemBackend) Exists(_ context.Context, key string) (bool, error) {
	path := f.keyToPath(key)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// keyToPath converts a storage key to a sharded filesystem path.
func (f *FilesystemBackend) keyToPath(key string) string {
	if len(key) >= 4 {
		return filepath.Join(f.basePath, key[:2], key[2:4], key)
	}
	return filepath.Join(f.basePath, key)
}
