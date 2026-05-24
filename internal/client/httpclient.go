package client

import (
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"resty.dev/v3"

	"github.com/suprsend/cli/internal/config"
)

// Options configures the resty client returned by NewHTTPClientWithOptions.
type Options struct {
	// Transport, if non-nil, is wired via resty.SetTransport. Used by
	// pkg/mcpserver consumers to inject an authExpiryTransport for reactive
	// session closure on 401.
	Transport http.RoundTripper
	// Timeout caps every request. Zero = no timeout (resty default).
	Timeout time.Duration
}

// NewHTTPClient is the legacy CLI entry point — equivalent to
// NewHTTPClientWithOptions(Options{}).
func NewHTTPClient() *resty.Client {
	return NewHTTPClientWithOptions(Options{})
}

// NewHTTPClientWithOptions returns a resty.Client honoring the supplied
// transport + timeout. Preserves the existing proxy-config behavior.
func NewHTTPClientWithOptions(opts Options) *resty.Client {
	rc := resty.New()
	if opts.Transport != nil {
		rc.SetTransport(opts.Transport)
	}
	if opts.Timeout > 0 {
		rc.SetTimeout(opts.Timeout)
	}
	if proxyURL := config.Cfg.ProxyURL.String(); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			log.WithError(err).Error("Invalid HTTP_PROXY")
			return rc
		}
		rc.SetProxy(parsed.String())
	}
	return rc
}
