package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kubarax/assets/config"
	"kubarax/assets/envmap"
	"kubarax/templates"
	"kubarax/utils"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// GenerateOptions holds resolved options for the generate command
type GenerateOptions struct {
	templateType        templates.TemplateType
	dryRun              bool
	cwd                 string
	configFilePath      string
	dotEnvFilePath      string
	envVarPrefix        string
	managedCatalogPath  string
	customerCatalogPath string
}

// GenerateFlags holds raw CLI flags
type GenerateFlags struct {
	TerraformFlag       bool
	HelmFlag            bool
	DryRunFlag          bool
	ManagedCatalogPath  string
	CustomerCatalogPath string
	EnvPrefixFlag       string
}

// NewGenerateFlags returns default generate flags
func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{
		ManagedCatalogPath:  templates.DefaultManagedCatalogPath,
		CustomerCatalogPath: templates.DefaultCustomerCatalogPath,
		EnvPrefixFlag:       "KUBARAX_",
	}
}

// NewGenerateCmd creates the generate CLI command
func NewGenerateCmd() *cli.Command {
	flags := NewGenerateFlags()
	cmd := &cli.Command{
		Name:  "generate",
		Usage: "Generate FluxCD manifests from templates and configuration",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return err
			}
			return o.Run()
		},
	}

	flags.AddFlags(cmd)
	return cmd
}

// ToOptions converts CLI flags to resolved GenerateOptions
func (flags *GenerateFlags) ToOptions(cmd *cli.Command) (*GenerateOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, err
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, err
	}
	dotEnvFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, err
	}

	templateType := templates.TemplateTypeAll
	if flags.TerraformFlag && !flags.HelmFlag {
		templateType = templates.TemplateTypeTerraform
	} else if flags.HelmFlag && !flags.TerraformFlag {
		templateType = templates.TemplateTypeHelm
	}

	managedPath := flags.ManagedCatalogPath
	if !filepath.IsAbs(managedPath) {
		managedPath = filepath.Join(cwd, managedPath)
	}

	customerPath := flags.CustomerCatalogPath
	if !filepath.IsAbs(customerPath) {
		customerPath = filepath.Join(cwd, customerPath)
	}

	return &GenerateOptions{
		templateType:        templateType,
		dryRun:              flags.DryRunFlag,
		cwd:                 cwd,
		configFilePath:      configFilePath,
		dotEnvFilePath:      dotEnvFilePath,
		envVarPrefix:        flags.EnvPrefixFlag,
		managedCatalogPath:  managedPath,
		customerCatalogPath: customerPath,
	}, nil
}

// AddFlags registers generate-specific flags
func (flags *GenerateFlags) AddFlags(cmd *cli.Command) {
	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "terraform",
			Usage:       "Generate only Terraform files",
			Destination: &flags.TerraformFlag,
		},
		&cli.BoolFlag{
			Name:        "helm",
			Usage:       "Generate only Helm/FluxCD files",
			Destination: &flags.HelmFlag,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "Preview generated files without writing",
			Destination: &flags.DryRunFlag,
		},
		&cli.StringFlag{
			Name:        "managed-catalog",
			Value:       flags.ManagedCatalogPath,
			Usage:       "Path to managed service catalog output",
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "customer-catalog",
			Value:       flags.CustomerCatalogPath,
			Usage:       "Path to customer service catalog output",
			Destination: &flags.CustomerCatalogPath,
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for environment variables",
			Destination: &flags.EnvPrefixFlag,
		},
	}
}

// Run executes the generate command
func (o *GenerateOptions) Run() error {
	// Process clusters and generate template results
	results, clusterNames, err := o.processClusters()
	if err != nil {
		return err
	}

	// Clean up old generated files
	if err := o.cleanupOldFiles(clusterNames); err != nil {
		return err
	}

	// Write generated files
	if err := o.writeTemplateResults(results); err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	log.Info().Msgf("%s Generated %d files for %d cluster(s)", green("✓"), len(results), len(clusterNames))
	return nil
}

// processClusters loads config and generates templates for all clusters
func (o *GenerateOptions) processClusters() ([]templates.TemplateResult, []string, error) {
	// Load env
	em := envmap.NewEnvMapManager(o.dotEnvFilePath, ".", o.envVarPrefix)
	if err := em.Load(); err != nil {
		log.Warn().Msgf("Could not load env file: %s", err)
	}

	// Load config
	cm := config.NewConfigManager(o.configFilePath)
	if err := cm.Load(); err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	if err := cm.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validating config: %w", err)
	}

	var allResults []templates.TemplateResult
	var clusterNames []string

	for _, cluster := range cm.GetConfig().Clusters {
		clusterNames = append(clusterNames, cluster.Name)

		// Build template context from cluster config
		data, err := buildTemplateContext(cluster)
		if err != nil {
			return nil, nil, fmt.Errorf("building template context for %s: %w", cluster.Name, err)
		}

		// Add env map to context
		data["env"] = map[string]interface{}{
			"fluxGitHTTPSUrl": em.GetConfig().FluxGitHTTPSUrl,
			"fluxGitUsername": em.GetConfig().FluxGitUsername,
			"domainName":      em.GetConfig().DomainName,
		}

		// Generate templates
		results, err := templates.TemplateAllFiles(o.templateType, data)
		if err != nil {
			log.Warn().Msgf("Some templates had errors for cluster %s: %s", cluster.Name, err)
		}

		// Resolve output paths
		for i := range results {
			results[i].Path = o.resolveOutputPath(results[i].Path, cluster.Name)
		}

		allResults = append(allResults, results...)
	}

	return allResults, clusterNames, nil
}

// buildTemplateContext converts a Cluster config to a template-friendly map
func buildTemplateContext(cluster config.Cluster) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("marshaling cluster config: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling cluster config: %w", err)
	}

	return data, nil
}

// resolveOutputPath transforms template paths into output paths
func (o *GenerateOptions) resolveOutputPath(templatePath, clusterName string) string {
	// Remove the "embedded/" prefix
	path := strings.TrimPrefix(templatePath, "embedded/")

	// Replace "example" directory placeholder with cluster name
	path = strings.ReplaceAll(path, "/example/", "/"+clusterName+"/")

	// Remove .tplt extension
	path = strings.TrimSuffix(path, ".tplt")

	// Determine output base directory
	if strings.HasPrefix(path, "managed-service-catalog/") {
		return filepath.Join(o.managedCatalogPath, strings.TrimPrefix(path, "managed-service-catalog/"))
	}
	if strings.HasPrefix(path, "customer-service-catalog/") {
		return filepath.Join(o.customerCatalogPath, strings.TrimPrefix(path, "customer-service-catalog/"))
	}

	return filepath.Join(o.cwd, path)
}

// cleanupOldFiles removes previously generated directories
func (o *GenerateOptions) cleanupOldFiles(clusterNames []string) error {
	if o.dryRun {
		log.Info().Msg("[DRY-RUN] Would clean up old generated files")
		return nil
	}

	for _, name := range clusterNames {
		managedPath := filepath.Join(o.managedCatalogPath, "helm", name)
		if err := os.RemoveAll(managedPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Msgf("Could not clean up %s: %s", managedPath, err)
		}

		customerPath := filepath.Join(o.customerCatalogPath, "helm", name)
		if err := os.RemoveAll(customerPath); err != nil && !os.IsNotExist(err) {
			log.Warn().Msgf("Could not clean up %s: %s", customerPath, err)
		}
	}

	return nil
}

// writeTemplateResults writes generated template files to disk
func (o *GenerateOptions) writeTemplateResults(results []templates.TemplateResult) error {
	for _, result := range results {
		// Skip files that rendered to empty/whitespace-only content
		if len(strings.TrimSpace(string(result.Content))) == 0 {
			log.Debug().Msgf("Skipped empty file: %s", result.Path)
			continue
		}

		if o.dryRun {
			log.Info().Msgf("[DRY-RUN] Would write: %s", result.Path)
			continue
		}

		dir := filepath.Dir(result.Path)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		if err := os.WriteFile(result.Path, result.Content, 0644); err != nil {
			return fmt.Errorf("writing file %s: %w", result.Path, err)
		}

		log.Debug().Msgf("Wrote: %s", result.Path)
	}

	return nil
}
