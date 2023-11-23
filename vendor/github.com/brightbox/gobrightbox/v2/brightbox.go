package brightbox

import (
	"context"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// Oauth2 is the abstract interface for any Brightbox oauth2 client generator
type Oauth2 interface {
	Client(ctx context.Context) (*http.Client, oauth2.TokenSource, error)
	APIURL() (*url.URL, error)
}

// Connect allocates and configures a Client for interacting with the API.
func Connect(ctx context.Context, config Oauth2) (*Client, error) {
	baseURL, err := config.APIURL()
	if err != nil {
		return nil, err
	}
	httpClient, tokenSource, err := config.Client(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:     baseURL,
		client:      httpClient,
		tokensource: tokenSource,
	}, nil
}
