package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"kubarax/assets/config"
	"kubarax/assets/envmap"
	"kubarax/internal/bootstrap"
	"kubarax/templates"
	"kubarax/utils"

	"github.com/urfave/cli/v3"
)

// BootstrapFlags holds raw CLI flags for the bootstrap command
type BootstrapFlags struct {
	WithES                 bool
	WithProm               bool
	ClusterSecretStorePath string
	ManagedCatalogPath     string
	OverlayValuesPath      string
	EnvFile                string
	EnvPrefixFlag          string
	DryRun                 bool
	Timeout                time.Duration
	ClusterName            string
}

// NewBootstrapFlags returns default bootstrap flags
func NewBootstrapFlags() *BootstrapFlags {
	return &BootstrapFlags{
		WithES:        true,
		WithProm:      true,
		EnvFile:       ".env",
		EnvPrefixFlag: "KUBARAX_",
		Timeout:       5 * time.Minute,
	}
}

// NewBootstrapCmd creates the bootstrap CLI command
func NewBootstrapCmd() *cli.Command {
	flags := NewBootstrapFlags()
	var clusterNameArg string
	cmd := &cli.Command{
		Name:      "bootstrap",
		Usage:     "Bootstrap FluxCD onto the specified cluster with optional external-secrets and prometheus CRDs",
		ArgsUsage: "<cluster-name>",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "cluster-name",
				UsageText:   "The name of the cluster as set in the config",
				Destination: &clusterNameArg,
				Min:         1,
				Max:         1,
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			if clusterNameArg == "" {
				return fmt.Errorf("missing argument: cluster-name")
			}
			flags.ClusterName = clusterNameArg
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return fmt.Errorf("couldn't convert flags to options: %w", err)
			}
			return runBootstrap(c, o)
		},
	}

	flags.AddFlags(cmd)
	return cmd
}

// ToOptions converts CLI flags to bootstrap Options
func (flags *BootstrapFlags) ToOptions(cmd *cli.Command) (*bootstrap.Options, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, err
	}

	envFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, err
	}

	kubeconf, err := utils.GetFullPath(cmd.String("kubeconfig"), cwd)
	if err != nil {
		return nil, err
	}

	managedAbsPath := flags.ManagedCatalogPath
	if !filepath.IsAbs(managedAbsPath) {
		managedAbsPath, err = filepath.Abs(filepath.Join(cwd, managedAbsPath))
		if err != nil {
			return nil, fmt.Errorf("getting absolute path failed: %w", err)
		}
	}

	customerAbsPath := flags.OverlayValuesPath
	if !filepath.IsAbs(customerAbsPath) {
		customerAbsPath, err = filepath.Abs(filepath.Join(cwd, customerAbsPath))
		if err != nil {
			return nil, fmt.Errorf("getting absolute path failed: %w", err)
		}
	}

	// Load environment
	em := envmap.NewEnvMapManager(envFilePath, ".", flags.EnvPrefixFlag)
	if err := em.Load(); err != nil {
		return nil, fmt.Errorf("reading env failed: %w", err)
	}
	if err := em.ValidateAll(); err != nil {
		return nil, fmt.Errorf("validating env: %w", err)
	}

	envMap := em.GetConfig()

	// Load config file
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("getting config file path: %w", err)
	}

	cm := config.NewConfigManager(configFilePath)
	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("loading config from %s: %w", configFilePath, err)
	}
	if err := cm.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Find the cluster by name
	clusterName := flags.ClusterName
	var clusterConfig *config.Cluster
	for i := range cm.GetConfig().Clusters {
		if cm.GetConfig().Clusters[i].Name == clusterName {
			clusterConfig = &cm.GetConfig().Clusters[i]
			break
		}
	}
	if clusterConfig == nil {
		return nil, fmt.Errorf("cluster '%s' not found in config file %s", clusterName, configFilePath)
	}

	// Validate ClusterSecretStore path if provided
	var cssAbsPath string
	if flags.ClusterSecretStorePath != "" {
		if !filepath.IsAbs(flags.ClusterSecretStorePath) {
			cssAbsPath, err = filepath.Abs(filepath.Join(cwd, flags.ClusterSecretStorePath))
			if err != nil {
				return nil, fmt.Errorf("getting absolute path for ClusterSecretStore file: %w", err)
			}
		} else {
			cssAbsPath = flags.ClusterSecretStorePath
		}

		if _, err := os.Stat(cssAbsPath); err != nil {
			return nil, fmt.Errorf("ClusterSecretStore file not found: %w", err)
		}
	}

	return &bootstrap.Options{
		Kubeconfig:     kubeconf,
		ManagedCatalog: managedAbsPath,
		OverlayValues:  customerAbsPath,
		WithES:         flags.WithES,
		WithProm:       flags.WithProm,
		WithESCSSPath:  cssAbsPath,
		EnvMap:         envMap,
		ClusterConfig:  clusterConfig,
		DryRun:         flags.DryRun,
		Timeout:        flags.Timeout,
		ClusterName:    clusterName,
	}, nil
}

// AddFlags registers bootstrap-specific flags
func (flags *BootstrapFlags) AddFlags(cmd *cli.Command) {
	bootstrapFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "dry-run",
			Value:       false,
			Usage:       "Run in dry-run mode",
			Destination: &flags.DryRun,
		},
		&cli.BoolFlag{
			Name:        "with-es-crds",
			Usage:       "Also install external-secrets CRDs",
			Destination: &flags.WithES,
		},
		&cli.BoolFlag{
			Name:        "with-prometheus-crds",
			Usage:       "Also install kube-prometheus-stack CRDs",
			Destination: &flags.WithProm,
		},
		&cli.StringFlag{
			Name:        "with-es-css-file",
			Usage:       "Path to ClusterSecretStore manifest file (supports go-template + sprig)",
			Destination: &flags.ClusterSecretStorePath,
		},
		&cli.StringFlag{
			Name:        "managed-catalog",
			Value:       templates.DefaultManagedCatalogPath,
			Usage:       "Path to managed catalog directory",
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Value:       templates.DefaultCustomerCatalogPath,
			Usage:       "Path to overlay values directory",
			Destination: &flags.OverlayValuesPath,
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for environment variables",
			Destination: &flags.EnvPrefixFlag,
		},
		&cli.DurationFlag{
			Name:        "timeout",
			Value:       5 * time.Minute,
			Usage:       "Timeout for kubernetes API calls (e.g. 10s, 1m)",
			Destination: &flags.Timeout,
		},
	}

	cmd.Flags = append(cmd.Flags, bootstrapFlags...)
}

func runBootstrap(ctx context.Context, o *bootstrap.Options) error {
	ctx, cancelSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancelSignal()

	return bootstrap.Bootstrap(ctx, o)
}
