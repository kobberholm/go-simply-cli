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
	products []sdk.Product
	err      error
}

func (f fakeProducts) List(context.Context) ([]sdk.Product, sdk.Response, error) {
	return f.products, sdk.Response{RateLimitLimit: "20", RateLimitRemaining: "19"}, f.err
}

type fakeClient struct {
	products simply.Products
	dns      simply.DNS
}

func (f fakeClient) Products() simply.Products { return f.products }
func (f fakeClient) DNS() simply.DNS {
	if f.dns == nil {
		return &fakeDNS{}
	}
	return f.dns
}

type fakeDNS struct {
	records []sdk.Record
	zone    sdk.Zone
	product string
}

func (f *fakeDNS) ListRecords(_ context.Context, product string) ([]sdk.Record, sdk.Response, error) {
	f.product = product
	return f.records, sdk.Response{RateLimitRemaining: "18"}, nil
}

func (f *fakeDNS) Zone(context.Context, string) (sdk.Zone, sdk.Response, error) {
	return f.zone, sdk.Response{RateLimitRemaining: "17"}, nil
}

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

func TestProductsListUsesProductService(t *testing.T) {
	var stdout, stderr bytes.Buffer
	deps := defaultDependencies()
	deps.newClient = func(simply.Config) (simply.Client, error) {
		return fakeClient{products: fakeProducts{products: []sdk.Product{{Object: "example.test", Type: "domain"}}}}, nil
	}
	err := runWithDependencies([]string{"--non-interactive", "--api-key", "key", "products", "list", "--output", "json"}, strings.NewReader(""), &stdout, &stderr, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"example.test"`) || !strings.Contains(stdout.String(), `"data"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRecordsListPassesProductToDNSService(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dns := &fakeDNS{records: []sdk.Record{{ID: "7", Name: "www", Type: "CNAME", Value: "target.example."}}}
	deps := defaultDependencies()
	deps.newClient = func(simply.Config) (simply.Client, error) {
		return fakeClient{products: fakeProducts{}, dns: dns}, nil
	}
	err := runWithDependencies([]string{"--non-interactive", "--api-key", "key", "dns", "records", "list", "--product", "example.test"}, strings.NewReader(""), &stdout, &stderr, deps)
	if err != nil {
		t.Fatal(err)
	}
	if dns.product != "example.test" || !strings.Contains(stdout.String(), "www") {
		t.Fatalf("product = %q, stdout = %q", dns.product, stdout.String())
	}
}

func TestZoneShowUsesDNSService(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dns := &fakeDNS{zone: sdk.Zone{Name: "example.test", Records: []sdk.Record{{Name: "www", Type: "A", Value: "192.0.2.1"}}}}
	deps := defaultDependencies()
	deps.newClient = func(simply.Config) (simply.Client, error) {
		return fakeClient{products: fakeProducts{}, dns: dns}, nil
	}
	err := runWithDependencies([]string{"--non-interactive", "--api-key", "key", "dns", "zone", "show", "--product", "example.test", "--output", "json"}, strings.NewReader(""), &stdout, &stderr, deps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"example.test"`) || !strings.Contains(stdout.String(), `"192.0.2.1"`) {
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
