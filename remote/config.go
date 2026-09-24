package remote

import (
	"net/http"
	"time"

	"github.com/domainry/domainry-todo-sdk/saashost"
)

type SourceAuthorizer = saashost.SourceAuthorizer
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
