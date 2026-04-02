package helm

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// TemplateOptions holds configuration for helm template rendering
type TemplateOptions struct {
	ReleaseName string
	ChartPath   string
	Namespace   string
	ValuesPaths []string
	APIVersions []string
	SetArgs     []string
}

// Template renders a Helm chart to YAML manifests
func Template(ctx context.Context, opts TemplateOptions) ([]byte, error) {
	args := []string{"template", opts.ReleaseName, opts.ChartPath}

	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}

	for _, valuesPath := range opts.ValuesPaths {
		args = append(args, "--values", valuesPath)
	}

	for _, apiVersion := range opts.APIVersions {
		args = append(args, "--api-versions", apiVersion)
	}

	for _, setArg := range opts.SetArgs {
		args = append(args, "--set", setArg)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("helm template: %s: %w", string(output), err)
	}

	log.Debug().Msgf("Rendered helm template: %s", opts.ReleaseName)
	return output, nil
}
