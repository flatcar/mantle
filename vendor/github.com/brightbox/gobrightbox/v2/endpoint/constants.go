package endpoint

import (
	"golang.org/x/oauth2"
)

// API constants.
const (
	DefaultBaseURL      = "https://api.gb1.brightbox.com/"
	DefaultVersion      = "1.0"
	DefaultOrbitBaseURL = "https://orbit.brightbox.com/"
	DefaultOrbitVersion = "v1"
)

// InfrastructureScope tokens are used to access the Brightbox API
var InfrastructureScope = []string{"infrastructure"}

// OrbitScope tokens restrict access to Orbit files only
var OrbitScope = []string{"orbit"}

// FullScope tokens allow access to both the API and Orbit
var FullScope = append(InfrastructureScope, OrbitScope...)

// Brightbox is the default oauth2 endpoint
// As Brightbox is a direct access API using oauth2 mechanisms there is
// no AuthURL. Everything is driven via the TokenURL.
var Brightbox = oauth2.Endpoint{
	TokenURL:  DefaultBaseURL + "token/",
	AuthStyle: oauth2.AuthStyleInHeader,
}
