package filesystem

import (
	"os"
)

type InMemoryFileSystem struct {
	files map[string]*InMemoryFile
}

func NewInMemoryFileSystem() *InMemoryFileSystem {
	return &InMemoryFileSystem{
		files: make(map[string]*InMemoryFile),
	}
}

// NewMemoryFileSystem is an alias for NewInMemoryFileSystem for consistency
func NewMemoryFileSystem() *InMemoryFileSystem {
	return NewInMemoryFileSystem()
}

var _ FileSystem = (*InMemoryFileSystem)(nil)

func (fs *InMemoryFileSystem) OpenFile(path string, flag int, perm uint32) (File, error) {
	if _, exists := fs.files[path]; !exists {
		fs.files[path] = &InMemoryFile{}
	}
	return fs.files[path], nil
}

func (fs *InMemoryFileSystem) MkdirAll(path string, perm uint32) error {
	// In-memory filesystem does not need to create directories explicitly.
	return nil
}

// Exists checks if a file exists in the in-memory filesystem
func (fs *InMemoryFileSystem) Exists(path string) bool {
	_, exists := fs.files[path]
	return exists
}

// ReadFile reads the content of a file in the in-memory filesystem
func (fs *InMemoryFileSystem) ReadFile(path string) ([]byte, error) {
	if file, exists := fs.files[path]; exists {
		return file.GetData(), nil
	}
	return nil, os.ErrNotExist
}

// WriteFile writes content to a file in the in-memory filesystem
func (fs *InMemoryFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if _, exists := fs.files[path]; !exists {
		fs.files[path] = &InMemoryFile{}
	}
	fs.files[path].data = make([]byte, len(data))
	copy(fs.files[path].data, data)
	return nil
}

type InMemoryFile struct {
	data []byte
}

func (f *InMemoryFile) Write(data []byte) (int, error) {
	f.data = append(f.data, data...)
	return len(data), nil
}

func (f *InMemoryFile) Close() error {
	// No action needed for in-memory file close.
	return nil
}

func (f *InMemoryFile) GetData() []byte {
	return f.data
}
