package envmap

import "fmt"

// EnvMap holds environment variable configuration for kubarax
type EnvMap struct {
	// Project identification
	ProjectName  string `env:"PROJECT_NAME" default:"<...>" yaml:"PROJECT_NAME"`
	ProjectStage string `env:"PROJECT_STAGE" default:"<...>" yaml:"PROJECT_STAGE"`

	// Docker registry credentials
	DockerconfigBase64 string `env:"DOCKERCONFIG_BASE64" default:"<...>" yaml:"DOCKERCONFIG_BASE64"`

	// Git repository credentials (for FluxCD GitRepository source)
	FluxGitHTTPSUrl    string `env:"FLUX_GIT_HTTPS_URL" default:"<...>" yaml:"FLUX_GIT_HTTPS_URL"`
	FluxGitUsername    string `env:"FLUX_GIT_USERNAME" default:"<...>" yaml:"FLUX_GIT_USERNAME"`
	FluxGitPatOrPassword string `env:"FLUX_GIT_PAT_OR_PASSWORD" default:"<...>" yaml:"FLUX_GIT_PAT_OR_PASSWORD"`

	// DNS
	DomainName string `env:"DOMAIN_NAME" default:"<...>" yaml:"DOMAIN_NAME"`

	// Optional Helm repository credentials
	HelmRepoUsername string `env:"HELM_REPO_USERNAME" default:"" yaml:"HELM_REPO_USERNAME"`
	HelmRepoPassword string `env:"HELM_REPO_PASSWORD" default:"" yaml:"HELM_REPO_PASSWORD"`
	HelmRepoURL      string `env:"HELM_REPO_URL" default:"" yaml:"HELM_REPO_URL"`

	// Weave GitOps UI password (replaces ArgoCD wizard password)
	WeaveGitopsPassword string `env:"WEAVE_GITOPS_PASSWORD" default:"<...>" yaml:"WEAVE_GITOPS_PASSWORD"`
}

// ErrorEnvMap is a custom error type for environment validation
type ErrorEnvMap struct {
	Message string
	Err     error
}

func (e *ErrorEnvMap) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ErrorEnvMap) Unwrap() error {
	return e.Err
}

// Validate checks that all required fields are configured
func (em *EnvMap) Validate() error {
	required := map[string]string{
		"PROJECT_NAME":           em.ProjectName,
		"PROJECT_STAGE":          em.ProjectStage,
		"FLUX_GIT_HTTPS_URL":     em.FluxGitHTTPSUrl,
		"FLUX_GIT_USERNAME":      em.FluxGitUsername,
		"FLUX_GIT_PAT_OR_PASSWORD": em.FluxGitPatOrPassword,
		"DOMAIN_NAME":            em.DomainName,
		"WEAVE_GITOPS_PASSWORD":  em.WeaveGitopsPassword,
	}

	for name, value := range required {
		if value == "" || value == "<...>" {
			return &ErrorEnvMap{
				Message: fmt.Sprintf("required environment variable %s is not set or has placeholder value", name),
			}
		}
	}

	return nil
}

// ValidateAll validates all fields including optional ones
func (em *EnvMap) ValidateAll() error {
	return em.Validate()
}

// IsConfiguredEnvValue checks if a value is user-configured (not placeholder)
func IsConfiguredEnvValue(value string) bool {
	return value != "" && value != "<...>"
}

// SetDefaults populates empty fields with default values from struct tags
func (em *EnvMap) SetDefaults() {
	if em.ProjectName == "" {
		em.ProjectName = "<...>"
	}
	if em.ProjectStage == "" {
		em.ProjectStage = "<...>"
	}
	if em.DockerconfigBase64 == "" {
		em.DockerconfigBase64 = "<...>"
	}
	if em.FluxGitHTTPSUrl == "" {
		em.FluxGitHTTPSUrl = "<...>"
	}
	if em.FluxGitUsername == "" {
		em.FluxGitUsername = "<...>"
	}
	if em.FluxGitPatOrPassword == "" {
		em.FluxGitPatOrPassword = "<...>"
	}
	if em.DomainName == "" {
		em.DomainName = "<...>"
	}
	if em.WeaveGitopsPassword == "" {
		em.WeaveGitopsPassword = "<...>"
	}
}

// GenerateEnvExample generates an example .env file content
func (em *EnvMap) GenerateEnvExample() ([]byte, error) {
	example := `# kubarax Environment Configuration
# Fill in all placeholder values before running 'kubarax init'

# Project identification
KUBARAX_PROJECT_NAME=<...>
KUBARAX_PROJECT_STAGE=<...>

# Docker registry credentials (base64 encoded dockerconfig)
KUBARAX_DOCKERCONFIG_BASE64=<...>

# Git repository for FluxCD (HTTPS)
KUBARAX_FLUX_GIT_HTTPS_URL=<...>
KUBARAX_FLUX_GIT_USERNAME=<...>
KUBARAX_FLUX_GIT_PAT_OR_PASSWORD=<...>

# DNS domain
KUBARAX_DOMAIN_NAME=<...>

# Weave GitOps UI password
KUBARAX_WEAVE_GITOPS_PASSWORD=<...>

# Optional: Helm OCI repository credentials
# KUBARAX_HELM_REPO_USERNAME=
# KUBARAX_HELM_REPO_PASSWORD=
# KUBARAX_HELM_REPO_URL=
`
	return []byte(example), nil
}
