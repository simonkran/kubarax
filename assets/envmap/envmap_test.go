package envmap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvMapValidateRequiredFields(t *testing.T) {
	em := &EnvMap{
		ProjectName:      "test",
		ProjectStage:     "dev",
		FluxGitHTTPSUrl:  "https://github.com/org/repo",
		FluxGitUsername:   "git",
		FluxGitPatOrPassword: "token123",
		DomainName:       "example.com",
	}

	err := em.Validate()
	assert.NoError(t, err)
}

func TestEnvMapValidateFailsOnPlaceholder(t *testing.T) {
	em := &EnvMap{
		ProjectName:      "<...>",
		ProjectStage:     "dev",
		FluxGitHTTPSUrl:  "https://github.com/org/repo",
		FluxGitUsername:   "git",
		FluxGitPatOrPassword: "token",
		DomainName:       "example.com",
	}

	err := em.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PROJECT_NAME")
}

func TestEnvMapValidateFailsOnEmpty(t *testing.T) {
	em := &EnvMap{
		ProjectName:  "test",
		ProjectStage: "",
		DomainName:   "example.com",
	}

	err := em.Validate()
	assert.Error(t, err)
}

func TestEnvMapSetDefaults(t *testing.T) {
	em := &EnvMap{}
	em.SetDefaults()

	assert.Equal(t, "<...>", em.ProjectName)
	assert.Equal(t, "<...>", em.ProjectStage)
	assert.Equal(t, "<...>", em.DockerconfigBase64)
	assert.Equal(t, "<...>", em.FluxGitHTTPSUrl)
	assert.Equal(t, "<...>", em.FluxGitUsername)
	assert.Equal(t, "<...>", em.FluxGitPatOrPassword)
	assert.Equal(t, "<...>", em.DomainName)
}

func TestIsConfiguredEnvValue(t *testing.T) {
	assert.True(t, IsConfiguredEnvValue("myvalue"))
	assert.False(t, IsConfiguredEnvValue(""))
	assert.False(t, IsConfiguredEnvValue("<...>"))
}

func TestEnvMapGenerateExample(t *testing.T) {
	em := &EnvMap{}
	data, err := em.GenerateEnvExample()
	require.NoError(t, err)
	assert.Contains(t, string(data), "KUBARAX_PROJECT_NAME")
	assert.Contains(t, string(data), "KUBARAX_FLUX_GIT_HTTPS_URL")
	assert.NotContains(t, string(data), "WEAVE_GITOPS")
}

func TestEnvMapManagerLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	envContent := `KUBARAX_PROJECT_NAME=my-project
KUBARAX_PROJECT_STAGE=staging
KUBARAX_FLUX_GIT_HTTPS_URL=https://github.com/org/repo
KUBARAX_FLUX_GIT_USERNAME=git
KUBARAX_FLUX_GIT_PAT_OR_PASSWORD=ghp_token
KUBARAX_DOMAIN_NAME=example.com
`
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	mgr := NewEnvMapManager(envPath, dir, "KUBARAX_")
	err = mgr.Load()
	require.NoError(t, err)

	cfg := mgr.GetConfig()
	assert.Equal(t, "my-project", cfg.ProjectName)
	assert.Equal(t, "staging", cfg.ProjectStage)
	assert.Equal(t, "https://github.com/org/repo", cfg.FluxGitHTTPSUrl)
	assert.Equal(t, "git", cfg.FluxGitUsername)
	assert.Equal(t, "ghp_token", cfg.FluxGitPatOrPassword)
	assert.Equal(t, "example.com", cfg.DomainName)
}

func TestEnvMapManagerLoadIgnoresComments(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	envContent := `# This is a comment
KUBARAX_PROJECT_NAME=test
# Another comment
KUBARAX_PROJECT_STAGE=dev
`
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	mgr := NewEnvMapManager(envPath, dir, "KUBARAX_")
	err = mgr.Load()
	require.NoError(t, err)

	assert.Equal(t, "test", mgr.GetConfig().ProjectName)
	assert.Equal(t, "dev", mgr.GetConfig().ProjectStage)
}

func TestEnvMapManagerLoadStripsQuotes(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	envContent := `KUBARAX_PROJECT_NAME="quoted-value"
KUBARAX_PROJECT_STAGE='single-quoted'
`
	err := os.WriteFile(envPath, []byte(envContent), 0600)
	require.NoError(t, err)

	mgr := NewEnvMapManager(envPath, dir, "KUBARAX_")
	err = mgr.Load()
	require.NoError(t, err)

	assert.Equal(t, "quoted-value", mgr.GetConfig().ProjectName)
	assert.Equal(t, "single-quoted", mgr.GetConfig().ProjectStage)
}

func TestEnvMapManagerMissingFile(t *testing.T) {
	mgr := NewEnvMapManager("/nonexistent/.env", ".", "KUBARAX_")
	err := mgr.Load()
	// Should not error — missing file is acceptable, env vars are the fallback
	assert.NoError(t, err)
}
