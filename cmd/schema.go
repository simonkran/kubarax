package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"kubarax/assets/config"
	"kubarax/utils"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// SchemaOptions holds resolved options for the schema command
type SchemaOptions struct {
	outputFilePath string
}

// SchemaFlags holds raw CLI flags
type SchemaFlags struct {
	OutputFlag string
}

// NewSchemaFlags returns default schema flags
func NewSchemaFlags() *SchemaFlags {
	return &SchemaFlags{
		OutputFlag: "config.schema.json",
	}
}

// NewSchemaCmd creates the schema CLI command
func NewSchemaCmd() *cli.Command {
	flags := NewSchemaFlags()
	cmd := &cli.Command{
		Name:      "schema",
		Usage:     "Generate JSON schema file for config structure",
		UsageText: "schema [--output]",
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

// ToOptions converts CLI flags to resolved SchemaOptions
func (flags *SchemaFlags) ToOptions(cmd *cli.Command) (*SchemaOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, err
	}
	outputFilePath, err := utils.GetFullPath(flags.OutputFlag, cwd)
	if err != nil {
		return nil, err
	}

	return &SchemaOptions{
		outputFilePath: outputFilePath,
	}, nil
}

// AddFlags registers schema-specific flags
func (flags *SchemaFlags) AddFlags(cmd *cli.Command) {
	cmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Value:       flags.OutputFlag,
			Usage:       "Output file path for the JSON schema",
			Destination: &flags.OutputFlag,
		},
	}
}

// Run executes the schema command
func (o *SchemaOptions) Run() error {
	schemaDoc, err := config.GenerateSchema()
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(o.outputFilePath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	schemaJSON, err := json.MarshalIndent(schemaDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	if err := os.WriteFile(o.outputFilePath, schemaJSON, 0600); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	log.Info().Msgf("Generated schema file: %s", o.outputFilePath)
	return nil
}
