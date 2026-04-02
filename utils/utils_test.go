package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeB64(t *testing.T) {
	result, err := DecodeB64("aGVsbG8=")
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestDecodeB64Invalid(t *testing.T) {
	_, err := DecodeB64("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestEncodeB64(t *testing.T) {
	result := EncodeB64("hello")
	assert.Equal(t, "aGVsbG8=", result)
}

func TestFileExist(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.txt")

	// File does not exist
	exists, err := FileExist(filePath)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create the file
	err = os.WriteFile(filePath, []byte("test"), 0600)
	require.NoError(t, err)

	// File exists
	exists, err = FileExist(filePath)
	require.NoError(t, err)
	assert.True(t, exists)

	// Directory should return false
	exists, err = FileExist(dir)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestFileExistRelativePath(t *testing.T) {
	_, err := FileExist("relative/path.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path must be absolute")
}

func TestGetFullPathAbsolute(t *testing.T) {
	result, err := GetFullPath("/absolute/path/file.txt", "/some/cwd")
	require.NoError(t, err)
	assert.Equal(t, "/absolute/path/file.txt", result)
}

func TestGetFullPathRelative(t *testing.T) {
	result, err := GetFullPath("config.yaml", "/home/user/project")
	require.NoError(t, err)
	assert.Equal(t, "/home/user/project/config.yaml", result)
}

func TestGetFullPathTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := GetFullPath("~/myfile.txt", "/some/cwd")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, "myfile.txt"), result)
}

func TestGetFullPathEmpty(t *testing.T) {
	_, err := GetFullPath("", "/some/cwd")
	assert.Error(t, err)
}

func TestAddGitignore(t *testing.T) {
	dir := t.TempDir()

	err := AddGitignore(dir)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	require.NoError(t, err)

	assert.Contains(t, string(content), "kubarax")
	assert.Contains(t, string(content), ".terraform/")
	assert.Contains(t, string(content), ".env")
}

func TestAddGitignoreMergesExisting(t *testing.T) {
	dir := t.TempDir()
	gitignorePath := filepath.Join(dir, ".gitignore")

	// Create existing .gitignore
	err := os.WriteFile(gitignorePath, []byte("node_modules/\n*.log\n"), 0644)
	require.NoError(t, err)

	err = AddGitignore(dir)
	require.NoError(t, err)

	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	// Should contain both existing and new patterns
	assert.Contains(t, string(content), "node_modules/")
	assert.Contains(t, string(content), "kubarax")
	assert.Contains(t, string(content), ".terraform/")
}

func TestMergeGitignoreNoDuplicates(t *testing.T) {
	existing := []string{"*.log", "kubarax", ".env"}
	additions := []string{"kubarax", ".env", "new-pattern"}

	result := mergeGitignoreLines(existing, additions)

	// Count occurrences of "kubarax"
	count := 0
	for _, line := range result {
		if line == "kubarax" {
			count++
		}
	}
	assert.Equal(t, 1, count, "kubarax should appear only once")
	assert.Contains(t, result, "new-pattern")
}
