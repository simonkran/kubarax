package helm

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// RepoOptions holds configuration for adding a Helm repository
type RepoOptions struct {
	Name string
	URL  string
}

// AddRepository adds a Helm repository using helm CLI
func AddRepository(ctx context.Context, opts RepoOptions) error {
	cmd := exec.CommandContext(ctx, "helm", "repo", "add", opts.Name, opts.URL, "--force-update")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm repo add %s: %s: %w", opts.Name, string(output), err)
	}
	log.Debug().Msgf("Added helm repo %s: %s", opts.Name, opts.URL)
	return nil
}

// UpdateRepository updates a specific Helm repository
func UpdateRepository(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "helm", "repo", "update", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm repo update %s: %s: %w", name, string(output), err)
	}
	return nil
}
