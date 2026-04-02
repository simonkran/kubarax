package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"kubarax/assets/app"
	"kubarax/assets/config"
	"kubarax/assets/envmap"
	"kubarax/utils"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// InitOptions holds the resolved options for the init command
type InitOptions struct {
	copyPrepFolder bool
	force          bool
	cwd            string
	configFilePath string
	dotEnvFilePath string
	envVarPrefix   string
}

// InitFlags holds the raw CLI flags
type InitFlags struct {
	PrepFlag      bool
	ForceFlag     bool
	EnvFileFlag   string
	EnvPrefixFlag string
}

// NewInitFlags returns default init flags
func NewInitFlags() *InitFlags {
	return &InitFlags{
		PrepFlag:      false,
		ForceFlag:     false,
		EnvFileFlag:   ".env",
		EnvPrefixFlag: "KUBARAX_",
	}
}

// NewInitCmd creates the init CLI command
func NewInitCmd() *cli.Command {
	flags := NewInitFlags()
	cmd := &cli.Command{
		Name:  "init",
		Usage: "Initialize a new kubarax directory",
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

// ToOptions converts CLI flags to resolved InitOptions
func (flags *InitFlags) ToOptions(cmd *cli.Command) (*InitOptions, error) {
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

	return &InitOptions{
		copyPrepFolder: flags.PrepFlag,
		force:          flags.ForceFlag,
		cwd:            cwd,
		configFilePath: configFilePath,
		dotEnvFilePath: dotEnvFilePath,
		envVarPrefix:   flags.EnvPrefixFlag,
	}, nil
}

// AddFlags registers init-specific flags on the command
func (flags *InitFlags) AddFlags(cmd *cli.Command) {
	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "prep",
			Value:       flags.PrepFlag,
			Usage:       "Generate .gitignore and example .env file",
			Destination: &flags.PrepFlag,
		},
		&cli.BoolFlag{
			Name:        "overwrite",
			Value:       flags.ForceFlag,
			Usage:       "Overwrite config if exists",
			Destination: &flags.ForceFlag,
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for environment variables",
			Destination: &flags.EnvPrefixFlag,
		},
	}
}

// Run executes the init command logic
func (o *InitOptions) Run() error {
	em := envmap.NewEnvMapManager(o.dotEnvFilePath, ".", o.envVarPrefix)
	cm := config.NewConfigManager(o.configFilePath)

	envLoadErr := em.Load()
	cnfLoadErr := cm.Load()
	envValidateErr := em.Validate()

	em.SetDefaults()

	if envLoadErr != nil {
		log.Error().Msgf("Reading env failed: %s", envLoadErr)
		return envLoadErr
	}

	// Prep mode: generate .gitignore and example .env
	if o.copyPrepFolder {
		if err := utils.AddGitignore(o.cwd); err != nil {
			return err
		}

		_, dotenvStatError := os.Stat(o.dotEnvFilePath)
		if dotenvStatError == nil {
			log.Info().Msgf("Skipping dotenv creation. File exists: %v", em.GetFilepath())
		} else if os.IsNotExist(dotenvStatError) {
			exampleEnvMap, err := em.GenerateEnvExample()
			if err != nil {
				return err
			}
			if err := os.WriteFile(o.dotEnvFilePath, exampleEnvMap, 0600); err != nil {
				return err
			}
			log.Info().Msgf("Generated dotenv in path: %v", em.GetFilepath())
		} else {
			return dotenvStatError
		}
		return nil
	}

	// Force mode: overwrite existing config
	if o.force {
		if envValidateErr != nil {
			return fmt.Errorf("error validating env: %w", envValidateErr)
		}

		if fileExist, _ := utils.FileExist(cm.GetFilepath()); fileExist {
			app.CreateOrUpdateClusterFromEnv(cm.GetConfig(), em.GetConfig())
		} else {
			return fmt.Errorf("error loading config file: %s", cnfLoadErr)
		}

		if err := cm.Validate(); err != nil {
			return fmt.Errorf("error validating config: %s", err)
		}
		if err := cm.SaveToFile(); err != nil {
			return fmt.Errorf("error writing config: %s", err)
		}
		log.Info().Msgf("Overwritten config file: %s", cm.GetFilepath())
		log.Info().Msg("Initialized successfully")
		return nil
	}

	// Normal mode: create config from env if it doesn't exist
	if fileExist, err := utils.FileExist(cm.GetFilepath()); fileExist {
		log.Info().Msg("Config file already exists. To overwrite existing variables from env: set flag \"--overwrite\"")
		if err := cm.Validate(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if envValidateErr != nil {
			log.Info().Msg("Env validation error. If you want to generate an example dotenv, pass the \"--prep\" flag.")
			return fmt.Errorf("error validating env: %w", envValidateErr)
		}
		newCluster := config.NewClusterFromEnv(em.GetConfig())
		cm.GetConfig().Clusters = []config.Cluster{newCluster}
		if err := cm.SaveToFile(); err != nil {
			return err
		}
		log.Info().Msgf("Generated config in path: %v", cm.GetFilepath())
		return nil
	}

	log.Info().Msg("Initialized successfully")
	return nil
}
