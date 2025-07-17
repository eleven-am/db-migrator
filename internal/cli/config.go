package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StormConfig represents the storm.yaml configuration structure
type StormConfig struct {
	Version string `yaml:"version"`
	Project string `yaml:"project"`

	Database struct {
		Driver         string `yaml:"driver"`
		URL            string `yaml:"url"`
		MaxConnections int    `yaml:"max_connections"`
	} `yaml:"database"`

	Models struct {
		Package string `yaml:"package"`
	} `yaml:"models"`

	Migrations struct {
		Directory string `yaml:"directory"`
		Table     string `yaml:"table"`
		AutoApply bool   `yaml:"auto_apply"`
	} `yaml:"migrations"`

	ORM struct {
		GenerateHooks bool `yaml:"generate_hooks"`
		GenerateTests bool `yaml:"generate_tests"`
		GenerateMocks bool `yaml:"generate_mocks"`
	} `yaml:"orm"`

	Schema struct {
		StrictMode       bool   `yaml:"strict_mode"`
		NamingConvention string `yaml:"naming_convention"`
	} `yaml:"schema"`
}

func LoadStormConfig(path string) (*StormConfig, error) {
	if path == "" {
		locations := []string{"storm.yaml", "storm.yml", ".storm.yaml", ".storm.yml"}
		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				path = loc
				break
			}
		}
		if path == "" {
			return nil, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config StormConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Database.Driver == "" {
		config.Database.Driver = "postgres"
	}
	if config.Database.MaxConnections == 0 {
		config.Database.MaxConnections = 25
	}
	if config.Models.Package == "" {
		config.Models.Package = "./models"
	}
	if config.Migrations.Directory == "" {
		config.Migrations.Directory = "./migrations"
	}
	if config.Migrations.Table == "" {
		config.Migrations.Table = "schema_migrations"
	}
	if config.Schema.NamingConvention == "" {
		config.Schema.NamingConvention = "snake_case"
	}

	return &config, nil
}

func GetConfigPath() string {
	if path := os.Getenv("STORM_CONFIG"); path != "" {
		return path
	}

	locations := []string{"storm.yaml", "storm.yml", ".storm.yaml", ".storm.yml"}
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}

func SaveStormConfig(config *StormConfig, path string) error {
	if path == "" {
		path = "storm.yaml"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
