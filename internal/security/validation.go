package security

import (
	"fmt"
	"os"
	"strings"

	"github.com/rgonzalez12/dbd-analytics/internal/log"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	SteamAPIKey      string
	RequiredEnvVars  []string
	SensitiveEnvVars []string
}

func ValidateEnvironment() error {
	config := SecurityConfig{
		RequiredEnvVars: []string{
			"STEAM_API_KEY",
		},
		SensitiveEnvVars: []string{
			"STEAM_API_KEY",
			"CACHE_EVICTION_TOKEN",
		},
	}

	// Check required environment variables
	for _, envVar := range config.RequiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			return fmt.Errorf("required environment variable %s is not set", envVar)
		}
	}

	// Validate Steam API key format
	steamKey := os.Getenv("STEAM_API_KEY")
	if steamKey != "" {
		if len(steamKey) != 32 {
			log.Warn("Steam API key length is not standard (expected 32 characters)",
				"actual_length", len(steamKey))
		}

		// Check if it contains only alphanumeric characters
		for _, char := range steamKey {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
				log.Warn("Steam API key contains non-alphanumeric characters")
				break
			}
		}
	}

	logSecurityAudit(config)

	return nil
}

func logSecurityAudit(config SecurityConfig) {
	log.Info("Security audit completed",
		"required_env_vars_count", len(config.RequiredEnvVars),
		"sensitive_env_vars_count", len(config.SensitiveEnvVars),
		"steam_api_key_configured", os.Getenv("STEAM_API_KEY") != "",
		"cache_eviction_token_configured", os.Getenv("CACHE_EVICTION_TOKEN") != "",
		"log_level", os.Getenv("LOG_LEVEL"),
		"port", os.Getenv("PORT"))

	var setEnvVars []string
	for _, envVar := range []string{"LOG_LEVEL", "PORT", "WORKDIR"} {
		if os.Getenv(envVar) != "" {
			setEnvVars = append(setEnvVars, envVar)
		}
	}

	if len(setEnvVars) > 0 {
		log.Info("Non-sensitive environment variables configured",
			"variables", strings.Join(setEnvVars, ", "))
	}
}
