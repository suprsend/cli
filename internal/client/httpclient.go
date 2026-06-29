package client

import (
	"net/url"

	log "github.com/sirupsen/logrus"
	"resty.dev/v3"

	"github.com/suprsend/cli/internal/config"
)

func NewHTTPClient() *resty.Client {
	client := resty.New()

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
