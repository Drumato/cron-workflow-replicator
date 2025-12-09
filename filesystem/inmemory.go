package filesystem

type InMemoryFileSystem struct {
	files map[string]*InMemoryFile
}

func NewInMemoryFileSystem() *InMemoryFileSystem {
	return &InMemoryFileSystem{
		files: make(map[string]*InMemoryFile),
	}
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
