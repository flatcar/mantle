// Package endpoint manages the API endpoint details
//
// The API uses an endpoint URL and a semantic version number. This
// structure manages variations on the defaults and provides a function
// to access the correct URL
package endpoint

import (
	"net/url"

	"golang.org/x/oauth2"
)

// Config contains the endpoint,version and account of the Brightbox API
//
// BaseURL should be an url of the form https://api.region.brightbox.com,
// e.g: https://api.gb1.brightbox.com. Leave empty to use the default.
//
// Account should be the identifier of the default account to be used with
// the API. Clients authenticated with Brightbox APIClient credentials are
// only ever associated with one single Account, so you can leave this empty for
// those. Client's authenticated with Brightbox User credentials can have access
// to multiple accounts, so this parameter should be provided.
//
// Version is the major and minor numbers of the version of the Brightbox API you
// wish to access. Leave blank to use the default "1.0"
//
// Scopes specify optional requested permissions. Leave blank to
// request a token that can be used for all Brightbox API operations. Use
// [InfrastructureScope] or [OrbitScope] to restrict the token to those areas.
type Config struct {
	BaseURL string
	Version string
	Account string
	Scopes  []string
}

// APIURL provides the base URL for accessing the API using the Config
// entries. Where entries are missing, library defaults will be used.
//
// If Account is set, then a query parameter will be added to the URL to
// reference that account.
//
// APIURL is part of the [brightbox.Oauth2] access interface.
func (c *Config) APIURL() (*url.URL, error) {
	var rawURL string
	var rawVersion string
	if c.BaseURL == "" {
		rawURL = DefaultBaseURL
	} else {
		rawURL = c.BaseURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if c.Version == "" {
		rawVersion = DefaultVersion
	} else {
		rawVersion = c.Version
	}
	u, err = u.Parse(rawVersion + "/")
	if err != nil {
		return nil, err
	}
	if c.Account != "" {
		v := url.Values{}
		v.Set("account_id", c.Account)
		u.RawQuery = v.Encode()
	}
	return u, nil
}

// StorageURL provides the base URL for accessing Orbit using the Config
// entries. Where entries are missing, library defaults will be used.
//
// If Account is set, then a query parameter will be added to the URL to
// reference that account.
func (c *Config) StorageURL() (string, error) {
	var rawURL string
	var path string
	if c.BaseURL == "" {
		rawURL = DefaultOrbitBaseURL
	} else {
		rawURL = c.BaseURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if c.Version == "" {
		path = DefaultOrbitVersion
	} else {
		path = c.Version
	}
	if c.Account != "" {
		path = path + "/" + c.Account
	}
	newURL, err := u.Parse(path + "/")
	if err != nil {
		return "", err
	}
	return newURL.String(), nil
}

// TokenURL provides the OAuth2 URL from the Config BaseURL entries. Where
// entries are missing, library defaults will be used.
func (c *Config) TokenURL() (string, error) {
	tokenConfig := &Config{
		BaseURL: c.BaseURL,
		Version: "token",
	}
	u, err := tokenConfig.APIURL()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Endpoint provides an oauth2 Endpoint from the Brightbox endpoint
// entries. Where entries are missing, library defaults will be used.
func (c *Config) Endpoint() (*oauth2.Endpoint, error) {
	tokenurl, err := c.TokenURL()
	if err != nil {
		return nil, err
	}
	return &oauth2.Endpoint{
		TokenURL:  tokenurl,
		AuthStyle: oauth2.AuthStyleInHeader,
	}, nil
}
