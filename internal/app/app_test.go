package app

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kobberholm/go-simply-cli/internal/input"
	"github.com/kobberholm/go-simply-cli/internal/simply"
	sdk "github.com/kobberholm/go-simply-sdk"
)

type fakeProducts struct {
	err error
}

func (f fakeProducts) List(context.Context) ([]sdk.Product, sdk.Response, error) {
	return nil, sdk.Response{RateLimitLimit: "20", RateLimitRemaining: "19"}, f.err
}

type fakeClient struct {
	products simply.Products
}

func (f fakeClient) Products() simply.Products { return f.products }

func TestHelpPathsDoNotNeedCredentials(t *testing.T) {
	paths := [][]string{
		{"--help"}, {"auth", "--help"}, {"auth", "check", "--help"},
		{"products", "--help"}, {"products", "list", "--help"},
		{"dns", "--help"}, {"dns", "records", "--help"},
		{"dns", "records", "list", "--help"}, {"dns", "records", "add", "--help"},
		{"dns", "records", "update", "--help"}, {"dns", "records", "delete", "--help"},
		{"dns", "zone", "--help"}, {"dns", "zone", "show", "--help"}, {"dns", "reload", "--help"},
	}
	for _, args := range paths {
		var stdout, stderr bytes.Buffer
		if err := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr); err != nil {
			t.Errorf("Run(%q) error = %v", args, err)
		}
	}
}

func TestRunRejectsContradictoryInteractionModes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"--interactive", "--non-interactive", "products", "list"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("Run() error = %v, want interaction mode conflict", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"missing"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Run() error = %v, want unknown command", err)
	}
}

func TestAuthCheckUsesProvidedCredentialsWithoutPrompting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := defaultDependencies()
	deps.newClient = func(credentials simply.Config) (simply.Client, error) {
		if credentials.APIKey != "flag-key" || credentials.AuthMode != "bearer" {
			t.Fatalf("credentials = %+v", credentials)
		}
		return fakeClient{products: fakeProducts{}}, nil
	}
	deps.secret = func(io.Reader, io.Writer, string) (string, error) {
		t.Fatal("secret prompt should not be called")
		return "", nil
	}
	err := runWithDependencies([]string{"--non-interactive", "--api-key", "flag-key", "auth", "check"}, strings.NewReader(""), &stdout, &stderr, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Credentials are valid") || strings.Contains(stdout.String(), "flag-key") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestAuthCheckPromptsForMissingAPIKey(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := defaultDependencies()
	deps.newClient = func(credentials simply.Config) (simply.Client, error) {
		if credentials.APIKey != "prompted-key" {
			t.Fatalf("credentials = %+v", credentials)
		}
		return fakeClient{products: fakeProducts{}}, nil
	}
	deps.secret = input.Line
	err := runWithDependencies([]string{"--interactive", "auth", "check"}, strings.NewReader("prompted-key\n"), &stdout, &stderr, deps)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "prompted-key") {
		t.Fatalf("stdout leaked API key: %q", stdout.String())
	}
}

func TestAuthCheckUnauthorizedRedactsAPIKey(t *testing.T) {
	secret := "secret-key"
	var stdout, stderr bytes.Buffer
	deps := defaultDependencies()
	deps.newClient = func(simply.Config) (simply.Client, error) {
		return fakeClient{products: fakeProducts{err: unauthorizedError{}}}, nil
	}
	err := runWithDependencies([]string{"--non-interactive", "--api-key", secret, "auth", "check"}, strings.NewReader(""), &stdout, &stderr, deps)
	if err == nil || !strings.Contains(err.Error(), "authentication failed") || strings.Contains(err.Error(), secret) {
		t.Fatalf("error = %v", err)
	}
}

type unauthorizedError struct{}

func (unauthorizedError) Error() string        { return "unauthorized" }
func (unauthorizedError) IsUnauthorized() bool { return true }

func runWithDependencies(args []string, stdin io.Reader, stdout, stderr io.Writer, deps dependencies) error {
	root := newRootCommand(context.Background(), stdin, stdout, stderr, deps)
	root.SetArgs(args)
	return root.Execute()
}
