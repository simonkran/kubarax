package helm

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
)

// DependencyOptions holds configuration for updating helm dependencies
type DependencyOptions struct {
	ChartPath string
	Timeout   time.Duration
}

// UpdateDependencies runs helm dependency update on a chart
func UpdateDependencies(ctx context.Context, opts DependencyOptions) error {
	cmd := exec.CommandContext(ctx, "helm", "dependency", "update", opts.ChartPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm dependency update in %s: %s: %w", opts.ChartPath, string(output), err)
	}
	log.Debug().Msgf("Updated dependencies for: %s", opts.ChartPath)
	return nil
}
