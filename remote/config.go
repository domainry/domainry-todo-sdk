package remote

import (
	"context"
	"net/http"
	"time"

	toolsdk "github.com/domainry/domainry-tools-sdk"
)

type SourceAuthorizer func(context.Context, string, toolsdk.Authority) error
type Config struct {
	Endpoint           string
	ServiceAccessToken string
	HTTPClient         *http.Client
	RequestTimeout     time.Duration
	MaxRequestBytes    int64
	MaxResponseBytes   int64
	AuthorizeSource    SourceAuthorizer
}

func normalizeConfig(v Config) Config {
	if v.RequestTimeout <= 0 {
		v.RequestTimeout = 20 * time.Second
	}
	if v.MaxRequestBytes <= 0 {
		v.MaxRequestBytes = 4 << 20
	}
	if v.MaxResponseBytes <= 0 {
		v.MaxResponseBytes = 4 << 20
	}
	return v
}
