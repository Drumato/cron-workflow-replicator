package filesystem

import (
	"io"
	"os"
)

type DefaultFileSystem struct{}

func NewDefaultFileSystem() *DefaultFileSystem {
	return &DefaultFileSystem{}
}

var _ FileSystem = (*DefaultFileSystem)(nil)

func (fs *DefaultFileSystem) OpenFile(path string, flag int, perm uint32) (File, error) {
	file, err := os.OpenFile(path, flag, os.FileMode(perm))
	if err != nil {
		return nil, err
	}
	return &DefaultFile{file: file}, nil
}

func (fs *DefaultFileSystem) MkdirAll(path string, perm uint32) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

// Exists checks if a file exists on the actual filesystem
func (fs *DefaultFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads the content of a file from the actual filesystem
func (fs *DefaultFileSystem) ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	return io.ReadAll(file)
}

// WriteFile writes content to a file on the actual filesystem
func (fs *DefaultFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

type DefaultFile struct {
	file *os.File
}

var _ File = (*DefaultFile)(nil)

func (f *DefaultFile) Write(data []byte) (int, error) {
	return f.file.Write(data)
}

func (f *DefaultFile) Close() error {
	return f.file.Close()
}
