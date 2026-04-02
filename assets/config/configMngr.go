package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/mitchellh/mapstructure"
	jsonschemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	yamlv3 "gopkg.in/yaml.v3"
)

// ConfigManager handles loading, validating, and saving configuration
type ConfigManager struct {
	filepath string
	config   Config
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(filepath string) *ConfigManager {
	return &ConfigManager{
		filepath: filepath,
		config:   Config{},
	}
}

// GetFilepath returns the config file path
func (cm *ConfigManager) GetFilepath() string {
	return cm.filepath
}

// GetConfig returns a pointer to the loaded config
func (cm *ConfigManager) GetConfig() *Config {
	return &cm.config
}

// Load reads the config file and unmarshals it
func (cm *ConfigManager) Load() error {
	k := koanf.New(".")

	if err := k.Load(file.Provider(cm.filepath), yaml.Parser()); err != nil {
		return fmt.Errorf("error loading config file %s: %w", cm.filepath, err)
	}

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           &cm.config,
		TagName:          "yaml",
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return fmt.Errorf("error creating decoder: %w", err)
	}

	if err := decoder.Decode(k.All()); err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	return nil
}

// Validate validates the config against its JSON schema
func (cm *ConfigManager) Validate() error {
	schemaDoc, err := GenerateSchema()
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	schemaJSON, err := json.Marshal(schemaDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	configJSON, err := json.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	schemaDoc2, err := jsonschemaValidator.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	compiler := jsonschemaValidator.NewCompiler()
	schemaURL := "config.json"
	if err := compiler.AddResource(schemaURL, schemaDoc2); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var configData interface{}
	if err := json.Unmarshal(configJSON, &configData); err != nil {
		return fmt.Errorf("failed to unmarshal config for validation: %w", err)
	}

	if err := schema.Validate(configData); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// SaveToFile writes the config to its YAML file
func (cm *ConfigManager) SaveToFile() error {
	if err := os.MkdirAll(filepath.Dir(cm.filepath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yamlv3.Marshal(cm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GenerateSchema generates a JSON schema from the Config type
func GenerateSchema() (*jsonschema.Schema, error) {
	r := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
	}
	schema := r.Reflect(&Config{})
	return schema, nil
}
