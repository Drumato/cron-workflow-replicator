package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFileSystem_OpenFile_ErrorScenarios(t *testing.T) {
	fs := NewDefaultFileSystem()

	tests := []struct {
		name        string
		path        string
		flag        int
		perm        uint32
		expectError bool
		errorType   error
	}{
		{
			name:        "nonexistent directory",
			path:        "/nonexistent/dir/file.txt",
			flag:        os.O_CREATE | os.O_WRONLY,
			perm:        0644,
			expectError: true,
			errorType:   os.ErrNotExist,
		},
		{
			name:        "invalid path characters",
			path:        "/invalid\x00path/file.txt",
			flag:        os.O_CREATE | os.O_WRONLY,
			perm:        0644,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := fs.OpenFile(tt.path, tt.flag, tt.perm)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, file)
				if tt.errorType != nil {
					assert.True(t, errors.Is(err, tt.errorType))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, file)
				if file != nil {
					require.NoError(t, file.Close())
				}
			}
		})
	}
}

func TestDefaultFileSystem_MkdirAll_ErrorScenarios(t *testing.T) {
	fs := NewDefaultFileSystem()

	tests := []struct {
		name        string
		setupPath   string
		path        string
		perm        uint32
		expectError bool
		cleanup     func()
	}{
		{
			name:        "invalid path with null character",
			path:        "/tmp/invalid\x00path",
			perm:        0755,
			expectError: true,
		},
		{
			name: "permission denied (file exists as regular file)",
			setupPath: func() string {
				// Create a temporary file that will block directory creation
				tmpFile, err := os.CreateTemp("", "test-file-*")
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())
				return tmpFile.Name()
			}(),
			path:        "", // will be set in test
			perm:        0755,
			expectError: true,
			cleanup: func() {
				// Will be set in test
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.path
			var cleanup func()

			if tt.setupPath != "" {
				testPath = tt.setupPath + "/subdir"
				cleanup = func() {
					if err := os.Remove(tt.setupPath); err != nil && !os.IsNotExist(err) {
						t.Logf("Warning: failed to cleanup %s: %v", tt.setupPath, err)
					}
				}
				defer cleanup()
			}

			err := fs.MkdirAll(testPath, tt.perm)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Clean up created directory
				if err := os.RemoveAll(testPath); err != nil {
					t.Logf("Warning: failed to cleanup %s: %v", testPath, err)
				}
			}

			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func TestDefaultFileSystem_ReadFile_ErrorScenarios(t *testing.T) {
	fs := NewDefaultFileSystem()

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name:        "nonexistent file",
			path:        "/nonexistent/file.txt",
			expectError: true,
			errorCheck: func(err error) bool {
				return errors.Is(err, os.ErrNotExist)
			},
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorCheck: func(err error) bool {
				return errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file")
			},
		},
		{
			name:        "invalid path with null character",
			path:        "/invalid\x00path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := fs.ReadFile(tt.path)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, data)
				if tt.errorCheck != nil {
					assert.True(t, tt.errorCheck(err))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
			}
		})
	}
}

func TestDefaultFileSystem_WriteFile_ErrorScenarios(t *testing.T) {
	fs := NewDefaultFileSystem()

	tests := []struct {
		name        string
		setupPath   string
		path        string
		data        []byte
		perm        os.FileMode
		expectError bool
		cleanup     func()
	}{
		{
			name:        "invalid path with null character",
			path:        "/invalid\x00path.txt",
			data:        []byte("test data"),
			perm:        0644,
			expectError: true,
		},
		{
			name:        "nonexistent directory",
			path:        "/nonexistent/deeply/nested/file.txt",
			data:        []byte("test data"),
			perm:        0644,
			expectError: true,
		},
		{
			name: "permission denied on directory",
			setupPath: func() string {
				// Create a read-only directory
				tmpDir, err := os.MkdirTemp("", "readonly-dir-*")
				require.NoError(t, err)
				err = os.Chmod(tmpDir, 0444) // read-only
				require.NoError(t, err)
				return tmpDir
			}(),
			path:        "", // will be set in test
			data:        []byte("test data"),
			perm:        0644,
			expectError: true,
			cleanup: func() {
				// Will be set in test
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.path
			var cleanup func()

			if tt.setupPath != "" {
				testPath = filepath.Join(tt.setupPath, "test-file.txt")
				cleanup = func() {
					if err := os.Chmod(tt.setupPath, 0755); err != nil {
						t.Logf("Warning: failed to restore permissions on %s: %v", tt.setupPath, err)
					}
					if err := os.RemoveAll(tt.setupPath); err != nil {
						t.Logf("Warning: failed to cleanup %s: %v", tt.setupPath, err)
					}
				}
				defer cleanup()
			}

			err := fs.WriteFile(testPath, tt.data, tt.perm)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify the file was written correctly
				readData, readErr := fs.ReadFile(testPath)
				assert.NoError(t, readErr)
				assert.Equal(t, tt.data, readData)
				// Clean up
				if err := os.Remove(testPath); err != nil && !os.IsNotExist(err) {
					t.Logf("Warning: failed to cleanup %s: %v", testPath, err)
				}
			}

			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func TestDefaultFileSystem_Exists_EdgeCases(t *testing.T) {
	fs := NewDefaultFileSystem()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "invalid path with null character",
			path:     "/invalid\x00path",
			expected: false,
		},
		{
			name:     "existing system directory",
			path:     "/tmp",
			expected: true,
		},
		{
			name:     "nonexistent path",
			path:     "/absolutely/nonexistent/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.Exists(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultFile_Write_ErrorScenarios(t *testing.T) {
	fs := NewDefaultFileSystem()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-write-*")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to cleanup %s: %v", tmpFile.Name(), err)
		}
	}()
	require.NoError(t, tmpFile.Close())

	// Test writing after close
	t.Run("write after close", func(t *testing.T) {
		file, err := fs.OpenFile(tmpFile.Name(), os.O_WRONLY, 0644)
		require.NoError(t, err)

		// Close the file
		err = file.Close()
		require.NoError(t, err)

		// Try to write after close - should fail
		_, err = file.Write([]byte("test data"))
		assert.Error(t, err)
	})
}

func TestDefaultFile_Close_Multiple(t *testing.T) {
	fs := NewDefaultFileSystem()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-close-*")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to cleanup %s: %v", tmpFile.Name(), err)
		}
	}()
	require.NoError(t, tmpFile.Close())

	t.Run("multiple close calls", func(t *testing.T) {
		file, err := fs.OpenFile(tmpFile.Name(), os.O_WRONLY, 0644)
		require.NoError(t, err)

		// First close should succeed
		err = file.Close()
		assert.NoError(t, err)

		// Second close should fail
		err = file.Close()
		assert.Error(t, err)
	})
}
