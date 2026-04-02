package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"kubarax/cmd"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var version = "dev"

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})

	app := &cli.Command{
		Name:    "kubarax",
		Usage:   "A framework and bootstrapping tool for building and operating a production-grade Kubernetes platform with FluxCD",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "kubeconfig",
				Value:   defaultKubeconfig(),
				Usage:   "Path to the kubeconfig file",
				Sources: cli.EnvVars("KUBECONFIG"),
			},
			&cli.StringFlag{
				Name:    "work-dir",
				Aliases: []string{"w"},
				Value:   ".",
				Usage:   "Working directory",
			},
			&cli.StringFlag{
				Name:    "config-file",
				Aliases: []string{"c"},
				Value:   "config.yaml",
				Usage:   "Path to configuration file",
			},
			&cli.StringFlag{
				Name:  "env-file",
				Value: ".env",
				Usage: "Path to environment file",
			},
		},
		Commands: []*cli.Command{
			cmd.NewInitCmd(),
			cmd.NewGenerateCmd(),
			cmd.NewBootstrapCmd(),
			cmd.NewSchemaCmd(),
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Println("kubarax - Kubernetes Platform Engineering with FluxCD")
			fmt.Println("Run 'kubarax --help' for usage information")
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application failed")
	}
}

func defaultKubeconfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.kube/config"
}
