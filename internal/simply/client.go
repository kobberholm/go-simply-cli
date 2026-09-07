package simply

import (
	"context"
	"io"
	"net/http"

	sdk "github.com/kobberholm/go-simply-sdk"
)

type Products interface {
	// List returns products visible to the authenticated account.
	List(context.Context) ([]sdk.Product, sdk.Response, error)
}

type Client interface {
	// Products returns the product service.
	Products() Products
	// DNS returns the DNS service.
	DNS() DNS
}

type DNS interface {
	// ListRecords returns records for a product/domain.
	ListRecords(context.Context, string) ([]sdk.Record, sdk.Response, error)
	// Zone returns the DNS zone for a product/domain.
	Zone(context.Context, string) (sdk.Zone, sdk.Response, error)
}

type sdkClient struct {
	products Products
	dns      DNS
}

// Products returns the SDK-backed product service.
func (c sdkClient) Products() Products { return c.products }

// DNS returns the SDK-backed DNS service.
func (c sdkClient) DNS() DNS { return c.dns }

type Config struct {
	APIKey, Account, AuthMode, BaseURL string
	HTTPClient                         *http.Client
	DebugWriter                        io.Writer
}

// NewClient constructs an SDK-backed client and optionally wraps its HTTP transport for debug logging.
func NewClient(config Config) (Client, error) {
	httpClient := config.HTTPClient
	if config.DebugWriter != nil {
		httpClient = newDebugHTTPClient(httpClient, config.DebugWriter)
	}
	client, err := sdk.NewClient(sdk.Config{
		APIKey:     config.APIKey,
		Account:    config.Account,
		AuthMode:   sdk.AuthMode(config.AuthMode),
		BaseURL:    config.BaseURL,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, err
	}
	return sdkClient{products: client.Products(), dns: client.DNS()}, nil
}
