package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	APIKey   string
	AuthMode string
	Account  string
}

type Values struct {
	APIKeyFlag, AuthModeFlag, AccountFlag string
	Lookup                                func(string) string
}

// Resolve applies flag-over-environment precedence and validates credentials.
func Resolve(values Values) (Config, error) {
	lookup := values.Lookup
	if lookup == nil {
		lookup = os.Getenv
	}
	resolved := Config{
		APIKey:   firstNonEmpty(values.APIKeyFlag, lookup("SIMPLY_API_KEY")),
		AuthMode: strings.ToLower(firstNonEmpty(values.AuthModeFlag, lookup("SIMPLY_AUTH_MODE"))),
		Account:  firstNonEmpty(values.AccountFlag, lookup("SIMPLY_ACCOUNT")),
	}
	if resolved.AuthMode == "" {
		resolved.AuthMode = "bearer"
	}
	if resolved.AuthMode != "bearer" && resolved.AuthMode != "basic" {
		return Config{}, fmt.Errorf("auth mode must be bearer or basic")
	}
	if resolved.APIKey == "" {
		return Config{}, fmt.Errorf("API key is required")
	}
	if resolved.AuthMode == "basic" && resolved.Account == "" {
		return Config{}, fmt.Errorf("account is required for basic authentication")
	}
	return resolved, nil
}

// firstNonEmpty returns the first value containing non-whitespace text.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
