package bootstrap

import (
	"context"
	"fmt"

	"kubarax/internal/k8s"

	"github.com/rs/zerolog/log"
)

// SecretManager handles creating bootstrap secrets
type SecretManager struct {
	client *k8s.Client
}

// NewSecretManager creates a new SecretManager
func NewSecretManager(client *k8s.Client) *SecretManager {
	return &SecretManager{client: client}
}

// CreateFluxGitSecret creates the Git credentials secret for FluxCD
func (sm *SecretManager) CreateFluxGitSecret(ctx context.Context, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would create flux-git-credentials secret")
		return nil
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: flux-git-credentials
  namespace: flux-system
type: Opaque
stringData:
  username: %s
  password: %s
`, opts.EnvMap.FluxGitUsername, opts.EnvMap.FluxGitPatOrPassword)

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-bootstrap-secrets"

	if err := sm.client.ApplyManifest(ctx, []byte(manifest), applyOpts); err != nil {
		return fmt.Errorf("applying flux git secret: %w", err)
	}

	log.Info().Msg("Created flux-git-credentials secret")
	return nil
}

// CreateDockerRegistrySecret creates the Docker registry credentials secret
func (sm *SecretManager) CreateDockerRegistrySecret(ctx context.Context, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would create docker-registry-credentials secret")
		return nil
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: docker-registry-credentials
  namespace: flux-system
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: %s
`, opts.EnvMap.DockerconfigBase64)

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-bootstrap-secrets"

	if err := sm.client.ApplyManifest(ctx, []byte(manifest), applyOpts); err != nil {
		return fmt.Errorf("applying docker registry secret: %w", err)
	}

	log.Info().Msg("Created docker-registry-credentials secret")
	return nil
}

// CreateHelmRepoSecret creates credentials for an OCI/HTTPS Helm repository
func (sm *SecretManager) CreateHelmRepoSecret(ctx context.Context, opts *Options) error {
	if opts.EnvMap.HelmRepoUsername == "" || opts.EnvMap.HelmRepoPassword == "" {
		return nil
	}

	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would create helm-repo-credentials secret")
		return nil
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: helm-repo-credentials
  namespace: flux-system
type: Opaque
stringData:
  username: %s
  password: %s
`, opts.EnvMap.HelmRepoUsername, opts.EnvMap.HelmRepoPassword)

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-bootstrap-secrets"

	if err := sm.client.ApplyManifest(ctx, []byte(manifest), applyOpts); err != nil {
		return fmt.Errorf("applying helm repo secret: %w", err)
	}

	log.Info().Msg("Created helm-repo-credentials secret")
	return nil
}
