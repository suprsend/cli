package client

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"resty.dev/v3"

	"github.com/suprsend/cli/internal/config"
)

func NewHTTPClient() *resty.Client {
	client := resty.New()
	configureRetries(client)

	if proxyURL := config.Cfg.ProxyURL.String(); proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			log.WithError(err).Error("Invalid HTTP_PROXY")
			return client
		}
		client.SetProxy(parsed.String())
	}

	return client
}

// configureRetries retries a request up to twice when the connection drops
// or the API answers 429/502/503/504. Every management API call the CLI makes
// is a read, an upsert or a commit, so repeating one is safe, POST and PATCH
// included: a repeated commit answers "no changes to commit".
func configureRetries(c *resty.Client) {
	c.SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond).
		SetRetryMaxWaitTime(3 * time.Second).
		SetRetryDefaultConditions(false).
		SetAllowNonIdempotentRetry(true).
		AddRetryConditions(shouldRetry).
		AddRetryHooks(func(res *resty.Response, err error) {
			if err != nil {
				log.Debugf("retrying after error: %v", err)
			} else if res != nil {
				log.Debugf("retrying after HTTP %d", res.StatusCode())
			}
		})
}

func shouldRetry(res *resty.Response, err error) bool {
	if err != nil {
		return isConnectionError(err)
	}
	if res == nil {
		return false
	}
	switch res.StatusCode() {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// isConnectionError reports whether err is a dropped or refused connection or
// a timeout, as opposed to a TLS, URL or cancellation error that a retry
// won't fix.
func isConnectionError(err error) bool {
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
