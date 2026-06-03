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
		// resty's SetProxy mutates the underlying *http.Transport's Proxy
		// field — it type-asserts the client transport to *http.Transport
		// (resty.dev/v3 Client.HTTPTransport) and, on failure, only logs at
		// resty's own logger level and silently returns without applying the
		// proxy. When a caller injects a custom transport (authExpiryTransport,
		// a plain http.RoundTripper, NOT an *http.Transport) the type assertion
		// fails, so rc.SetProxy would be a silent no-op and the config-based
		// proxy would be quietly dropped.
		//
		// We make that failure mode LOUD: a config-based proxy combined with an
		// injected transport cannot be applied here, because the proxy must be
		// configured on the *http.Transport that the custom transport wraps
		// (see internal/utils.authExpiryTransport.Base). Note that an injected
		// authExpiryTransport already wraps http.DefaultTransport, which honors
		// HTTP_PROXY / HTTPS_PROXY env vars via http.ProxyFromEnvironment — so
		// env-var proxies still work transparently; only config-file proxies
		// hit this limitation.
		if opts.Transport != nil {
			log.Warnf("Proxy %q from config cannot be applied to an injected HTTP transport; "+
				"set the proxy on the transport's base (or via HTTP_PROXY/HTTPS_PROXY env vars) instead", parsed.String())
		} else {
			rc.SetProxy(parsed.String())
		}
	}
	return rc
}
