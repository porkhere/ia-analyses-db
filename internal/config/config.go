package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	Database DatabaseConfig
	Athena   AthenaConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type AthenaConfig struct {
	Database       string
	Workgroup      string
	OutputLocation string
	AWSProfile     string
	Region         string
}

func Load() (AppConfig, error) {
	port, err := envInt("PGPORT", 54329)
	if err != nil {
		return AppConfig{}, err
	}

	cfg := AppConfig{
		Database: DatabaseConfig{
			Host:     envString("PGHOST", "127.0.0.1"),
			Port:     port,
			Name:     envString("PGDATABASE", "ia_analyses"),
			User:     envString("PGUSER", "ia_admin"),
			Password: envString("PGPASSWORD", "change_me"),
		},
		Athena: AthenaConfig{
			Database:       envString("ATHENA_DATABASE", "50lan_new"),
			Workgroup:      envString("ATHENA_WORKGROUP", "primary"),
			OutputLocation: envString("ATHENA_OUTPUT_LOCATION", "s3://50lan-athena-query-results/athena_results/"),
			AWSProfile:     envString("AWS_PROFILE", "default"),
			Region:         firstNonEmptyEnv([]string{"AWS_REGION", "AWS_DEFAULT_REGION"}, "ap-northeast-1"),
		},
	}

	return cfg, nil
}

func firstNonEmptyEnv(names []string, fallback string) string {
	for _, name := range names {
		value := strings.TrimSpace(os.Getenv(name))
		if value != "" {
			return value
		}
	}

	return fallback
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}

	return parsed, nil
}
