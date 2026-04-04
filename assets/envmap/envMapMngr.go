package envmap

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// EnvMapManager handles loading and managing environment configuration
type EnvMapManager struct {
	filepath  string
	cwd       string
	envPrefix string
	config    EnvMap
}

// NewEnvMapManager creates a new EnvMapManager
func NewEnvMapManager(filepath, cwd, envPrefix string) *EnvMapManager {
	return &EnvMapManager{
		filepath:  filepath,
		cwd:       cwd,
		envPrefix: envPrefix,
		config:    EnvMap{},
	}
}

// GetFilepath returns the env file path
func (em *EnvMapManager) GetFilepath() string {
	return em.filepath
}

// GetConfig returns a pointer to the loaded env config
func (em *EnvMapManager) GetConfig() *EnvMap {
	return &em.config
}

// Load reads the .env file and populates the EnvMap
func (em *EnvMapManager) Load() error {
	// First try to load from .env file
	if err := em.loadFromFile(); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("loading env file: %w", err)
		}
	}

	// Override with environment variables
	em.loadFromEnv()

	return nil
}

// loadFromFile reads key=value pairs from the .env file
func (em *EnvMapManager) loadFromFile() error {
	f, err := os.Open(em.filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove surrounding quotes
		value = strings.Trim(value, "\"'")

		em.setField(key, value)
	}

	return scanner.Err()
}

// loadFromEnv reads environment variables with the configured prefix
func (em *EnvMapManager) loadFromEnv() {
	envVars := os.Environ()
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		if strings.HasPrefix(key, em.envPrefix) {
			em.setField(key, value)
		}
	}
}

// setField sets a field in the EnvMap based on the environment variable key
func (em *EnvMapManager) setField(key, value string) {
	// Strip prefix for matching
	stripped := strings.TrimPrefix(key, em.envPrefix)

	switch stripped {
	case "PROJECT_NAME":
		em.config.ProjectName = value
	case "PROJECT_STAGE":
		em.config.ProjectStage = value
	case "DOCKERCONFIG_BASE64":
		em.config.DockerconfigBase64 = value
	case "FLUX_GIT_HTTPS_URL":
		em.config.FluxGitHTTPSUrl = value
	case "FLUX_GIT_USERNAME":
		em.config.FluxGitUsername = value
	case "FLUX_GIT_PAT_OR_PASSWORD":
		em.config.FluxGitPatOrPassword = value
	case "DOMAIN_NAME":
		em.config.DomainName = value
	case "HELM_REPO_USERNAME":
		em.config.HelmRepoUsername = value
	case "HELM_REPO_PASSWORD":
		em.config.HelmRepoPassword = value
	case "HELM_REPO_URL":
		em.config.HelmRepoURL = value
	case "ESS_VAULT_NAME":
		em.config.ESSVaultName = value
	case "ESS_SECRET_NAME":
		em.config.ESSSecretName = value
	case "ESS_TOKEN_KEY":
		em.config.ESSTokenKey = value
	case "ESS_TOKEN":
		em.config.ESSToken = value
	}
}

// Validate validates required env fields
func (em *EnvMapManager) Validate() error {
	return em.config.Validate()
}

// ValidateAll validates all fields
func (em *EnvMapManager) ValidateAll() error {
	return em.config.ValidateAll()
}

// SetDefaults sets default values for empty fields
func (em *EnvMapManager) SetDefaults() {
	em.config.SetDefaults()
}

// GenerateEnvExample generates an example .env file
func (em *EnvMapManager) GenerateEnvExample() ([]byte, error) {
	return em.config.GenerateEnvExample()
}

// GetEnvPrefix returns the configured prefix
func (em *EnvMapManager) GetEnvPrefix() string {
	return em.envPrefix
}

// String returns a summary of loaded configuration
func (em *EnvMapManager) String() string {
	return fmt.Sprintf("EnvMapManager{filepath=%s, prefix=%s}", em.filepath, em.envPrefix)
}
