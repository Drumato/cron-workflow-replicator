package filesystem

import (
	"os"
)

type FileSystem interface {
	OpenFile(path string, flag int, perm uint32) (File, error)
	MkdirAll(path string, perm uint32) error
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

type File interface {
	Write(data []byte) (int, error)
	Close() error
}
