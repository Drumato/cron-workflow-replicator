package filesystem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInMemoryFileSystem(t *testing.T) {
	fs := NewInMemoryFileSystem()
	assert.NotNil(t, fs)
	assert.NotNil(t, fs.files)
	assert.Equal(t, 0, len(fs.files))
}

func TestInMemoryFileSystem_OpenFile(t *testing.T) {
	fs := NewInMemoryFileSystem()

	// Test opening a new file
	file1, err := fs.OpenFile("test1.yaml", 0644, 0644)
	assert.NoError(t, err)
	assert.NotNil(t, file1)

	// Verify file was added to internal map
	assert.Equal(t, 1, len(fs.files))

	// Test opening the same file again should return the same file
	file2, err := fs.OpenFile("test1.yaml", 0644, 0644)
	assert.NoError(t, err)
	assert.Same(t, file1, file2)

	// Test opening a different file
	file3, err := fs.OpenFile("test2.yaml", 0644, 0644)
	assert.NoError(t, err)
	assert.NotSame(t, file1, file3)

	// Verify we now have 2 files
	assert.Equal(t, 2, len(fs.files))
}

func TestInMemoryFileSystem_MkdirAll(t *testing.T) {
	fs := NewInMemoryFileSystem()

	// MkdirAll should always succeed and do nothing
	err := fs.MkdirAll("/some/nested/path", 0755)
	assert.NoError(t, err)

	// Should not create any files
	assert.Equal(t, 0, len(fs.files))
}

func TestInMemoryFile_Write(t *testing.T) {
	fs := NewInMemoryFileSystem()
	file, _ := fs.OpenFile("test.yaml", 0644, 0644)

	// Test writing data
	data1 := []byte("hello ")
	n, err := file.Write(data1)
	assert.NoError(t, err)
	assert.Equal(t, len(data1), n)

	// Test appending more data
	data2 := []byte("world")
	n, err = file.Write(data2)
	assert.NoError(t, err)
	assert.Equal(t, len(data2), n)

	// Verify internal data
	memFile := file.(*InMemoryFile)
	expectedData := "hello world"
	assert.Equal(t, expectedData, string(memFile.data))
}

func TestInMemoryFile_Close(t *testing.T) {
	fs := NewInMemoryFileSystem()
	file, _ := fs.OpenFile("test.yaml", 0644, 0644)

	// Close should always succeed
	err := file.Close()
	assert.NoError(t, err)

	// Should be able to close multiple times
	err = file.Close()
	assert.NoError(t, err)
}

func TestInMemoryFileSystem_InterfaceCompliance(t *testing.T) {
	// This test ensures the type implements the interface
	var _ FileSystem = (*InMemoryFileSystem)(nil)
	var _ File = (*InMemoryFile)(nil)
}

func TestInMemoryFile_WriteMultipleFiles(t *testing.T) {
	fs := NewInMemoryFileSystem()

	// Create two files and write different content
	file1, _ := fs.OpenFile("file1.yaml", 0644, 0644)
	file2, _ := fs.OpenFile("file2.yaml", 0644, 0644)

	_, err := file1.Write([]byte("content1"))
	assert.NoError(t, err)
	_, err = file2.Write([]byte("content2"))
	assert.NoError(t, err)

	// Verify each file has its own content
	memFile1 := file1.(*InMemoryFile)
	memFile2 := file2.(*InMemoryFile)

	assert.Equal(t, "content1", string(memFile1.data))
	assert.Equal(t, "content2", string(memFile2.data))
}

func TestInMemoryFile_WriteEmptyData(t *testing.T) {
	fs := NewInMemoryFileSystem()
	file, _ := fs.OpenFile("test.yaml", 0644, 0644)

	// Test writing empty data
	n, err := file.Write([]byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	// Test writing nil data
	n, err = file.Write(nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}
