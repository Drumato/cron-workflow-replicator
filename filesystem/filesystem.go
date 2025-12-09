package filesystem

type FileSystem interface {
	OpenFile(path string, flag int, perm uint32) (File, error)
	MkdirAll(path string, perm uint32) error
}

type File interface {
	Write(data []byte) (int, error)
	Close() error
}
