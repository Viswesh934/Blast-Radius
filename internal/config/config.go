package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	OpenMetadata OpenMetadataConfig `mapstructure:"openmetadata"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Snapshot     SnapshotConfig     `mapstructure:"snapshot"`
	Output       OutputConfig       `mapstructure:"output"`
}

type OpenMetadataConfig struct {
	BaseURL  string `mapstructure:"baseurl"`
	JWTToken string `mapstructure:"jwt_token"`
}

type DatabaseConfig struct {
	FQN string `mapstructure:"fqn"`
}

type SnapshotConfig struct {
	Directory string `mapstructure:"directory"`
}

type OutputConfig struct {
	Format string `mapstructure:"format"`
}

func Load(cfgFile string) (*Config, error) {
	viper.Reset()

	viper.SetDefault("openmetadata.baseurl", "http://localhost:8585/api/v1")
	viper.SetDefault("openmetadata.jwt_token", "")
	viper.SetDefault("database.fqn", "")
	viper.SetDefault("snapshot.directory", "./snapshots")
	viper.SetDefault("output.format", "table")

	viper.SetEnvPrefix("BR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, err
		}
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.AddConfigPath(".")
		viper.SetConfigName(".blast-radius")
		_ = viper.ReadInConfig()
	}

	if envBaseURL := os.Getenv("OM_BASE_URL"); envBaseURL != "" {
		viper.Set("openmetadata.baseurl", envBaseURL)
	}
	if envToken := os.Getenv("OM_JWT_TOKEN"); envToken != "" {
		viper.Set("openmetadata.jwt_token", envToken)
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
