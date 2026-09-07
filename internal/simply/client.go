package simply

import (
	"context"
	"net/http"

	sdk "github.com/kobberholm/go-simply-sdk"
)

type Products interface {
	List(context.Context) ([]sdk.Product, sdk.Response, error)
}

type Client interface {
	Products() Products
}

type sdkClient struct {
	products Products
}

func (c sdkClient) Products() Products { return c.products }

type Config struct {
	APIKey, Account, AuthMode, BaseURL string
	HTTPClient                         *http.Client
}

func NewClient(config Config) (Client, error) {
	client, err := sdk.NewClient(sdk.Config{
		APIKey:     config.APIKey,
		Account:    config.Account,
		AuthMode:   sdk.AuthMode(config.AuthMode),
		BaseURL:    config.BaseURL,
		HTTPClient: config.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return sdkClient{products: client.Products()}, nil
}
