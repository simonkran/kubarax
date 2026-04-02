package utils

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DecodeB64 decodes a base64 string into raw string
func DecodeB64(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %w", err)
	}
	return string(decoded), nil
}

// EncodeB64 encodes a string to base64
func EncodeB64(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// FileExist checks if a file exists at the given path
func FileExist(path string) (bool, error) {
	if !filepath.IsAbs(path) {
		return false, fmt.Errorf("path must be absolute: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if info.IsDir() {
		return false, nil
	}

	return true, nil
}

// GetFullPath returns the absolute path representation of 'path'
func GetFullPath(path, cwd string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Handle home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	return filepath.Abs(filepath.Join(cwd, path))
}

// AddGitignore creates or merges a .gitignore file with kubarax patterns
func AddGitignore(cwd string) error {
	gitignorePath := filepath.Join(cwd, ".gitignore")

	// Resolve symlinks for security
	resolvedPath, err := resolveAndValidatePath(gitignorePath, cwd)
	if err != nil {
		return err
	}

	kubaraxPatterns := strings.Split(strings.TrimSpace(gitignoreKubarax), "\n")

	existingPatterns := []string{}
	if data, err := os.ReadFile(resolvedPath); err == nil {
		existingPatterns = strings.Split(string(data), "\n")
	}

	merged := mergeGitignoreLines(existingPatterns, kubaraxPatterns)

	return os.WriteFile(resolvedPath, []byte(strings.Join(merged, "\n")+"\n"), 0644)
}

// mergeGitignoreLines merges two gitignore pattern lists, removing duplicates
func mergeGitignoreLines(existing, additions []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, line := range existing {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			seen[trimmed] = true
		}
		result = append(result, line)
	}

	needsNewline := len(result) > 0 && result[len(result)-1] != ""

	for _, line := range additions {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || seen[trimmed] {
			continue
		}
		if needsNewline {
			result = append(result, "")
			needsNewline = false
		}
		result = append(result, line)
		seen[trimmed] = true
	}

	return result
}

// resolveAndValidatePath resolves symlinks and validates path is within cwd
func resolveAndValidatePath(path, cwd string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	// If file exists, resolve symlinks
	if _, err := os.Lstat(absPath); err == nil {
		resolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", fmt.Errorf("cannot resolve symlinks: %w", err)
		}
		absPath = resolved
	}

	// Ensure it stays within cwd
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("cannot resolve cwd: %w", err)
	}

	if !strings.HasPrefix(absPath, absCwd) {
		return "", fmt.Errorf("path escapes working directory: %s", absPath)
	}

	return absPath, nil
}

const gitignoreKubarax = `# kubarax
kubarax
kubarax.exe

# Terraform
.terraform/
*.tfstate
*.tfstate.backup
*.tfvars
.terraform.lock.hcl

# Helm
charts/
Chart.lock

# Secrets
.env
*.pem
*.key
credentials.json

# Logs
*.log

# OS
.DS_Store
Thumbs.db

# Archives
*.tar.gz
*.zip
`
