package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type CortensorConfig struct {
	RouterEndpoint     string `mapstructure:"router_endpoint"`
	SessionID          string `mapstructure:"session_id"`
	ConsensusThreshold int    `mapstructure:"consensus_threshold"`
}

type InferenceConfig struct {
	ActiveProvider string          `mapstructure:"active_provider"`
	Cortensor      CortensorConfig `mapstructure:"cortensor"`
}

type Config struct {
	Inference InferenceConfig `mapstructure:"inference"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	
	// Set default values
	v.SetDefault("inference.active_provider", "vibe")
	v.SetDefault("inference.cortensor.router_endpoint", "https://router.cortensor.io")
	v.SetDefault("inference.cortensor.consensus_threshold", 1)

	// Config file locations
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(DataDir())
	v.AddConfigPath(".")

	// Environment variables
	v.SetEnvPrefix("AURACRAB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is fine, we use defaults/env
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand environment variables in string fields (e.g., ${CORTENSOR_SESSION_ID})
	cfg.Inference.Cortensor.SessionID = os.ExpandEnv(cfg.Inference.Cortensor.SessionID)

	return &cfg, nil
}

func ConfigPath() string {
	return filepath.Join(DataDir(), "config.yaml")
}
