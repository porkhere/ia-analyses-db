package postgres

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"ia-analyses-db/internal/config"
)

func BuildDSN(cfg config.DatabaseConfig) string {
	return buildURL(cfg, false)
}

func RedactedDSN(cfg config.DatabaseConfig) string {
	return buildURL(cfg, true)
}

func IsLocalDatabaseTarget(cfg config.DatabaseConfig) bool {
	host := strings.TrimSpace(strings.ToLower(cfg.Host))
	if host == "" {
		return false
	}

	if host == "localhost" {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

func buildURL(cfg config.DatabaseConfig, redactPassword bool) string {
	password := cfg.Password
	if redactPassword {
		password = "*****"
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}

	query := url.Values{}
	query.Set("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	return dsn.String()
}
