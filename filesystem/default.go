package filesystem

import (
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
