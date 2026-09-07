package config

import "testing"

func TestResolveFlagPrecedenceAndDefaults(t *testing.T) {
	lookup := func(name string) string {
		values := map[string]string{
			"SIMPLY_API_KEY":   "environment-key",
			"SIMPLY_AUTH_MODE": "basic",
			"SIMPLY_ACCOUNT":   "environment-account",
		}
		return values[name]
	}
	resolved, err := Resolve(Values{APIKeyFlag: "flag-key", AccountFlag: "flag-account", Lookup: lookup})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.APIKey != "flag-key" || resolved.Account != "flag-account" || resolved.AuthMode != "basic" {
		t.Fatalf("resolved = %+v", resolved)
	}
}

func TestResolveValidatesBasicAccount(t *testing.T) {
	_, err := Resolve(Values{APIKeyFlag: "key", AuthModeFlag: "basic"})
	if err == nil || err.Error() != "account is required for basic authentication" {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestResolveDoesNotEchoAPIKey(t *testing.T) {
	_, err := Resolve(Values{APIKeyFlag: "secret-key", AuthModeFlag: "invalid"})
	if err == nil || err.Error() == "secret-key" {
		t.Fatalf("Resolve() error = %v", err)
	}
}
