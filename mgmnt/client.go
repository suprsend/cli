package mgmnt

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/client"
	suprsend "github.com/suprsend/suprsend-go"
	"resty.dev/v3"
)

func normalizeURL(url string) string {
	if url == "" {
		return url
	}
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}

type SS_MgmntClient struct {
	serviceToken     string
	hub_base_URL     string
	mgmnt_base_URL   string
	workspaceClients map[string]*suprsend.Client
	transport        http.RoundTripper
	debug            bool
}

func NewClientWithUrls(serviceToken string, baseURL string, mgmntURL string, debug bool) *SS_MgmntClient {
	// if service token is not set, log error and exit
	if serviceToken == "" {
		log.Fatal("Service token is required")
	}

	// custom > env > default
	if baseURL == "" {
		if envUrl := os.Getenv("SUPRSEND_BASE_URL"); envUrl != "" {
			baseURL = envUrl
		} else {
			baseURL = "https://hub.suprsend.com"
		}
	}

	// custom > env > default
	if mgmntURL == "" {
		if envUrl := os.Getenv("SUPRSEND_MGMNT_URL"); envUrl != "" {
			mgmntURL = envUrl
		} else {
			mgmntURL = "https://management-api.suprsend.com/"
		}
	}

	baseURL = normalizeURL(baseURL)
	mgmntURL = normalizeURL(mgmntURL)
	client := &SS_MgmntClient{
		serviceToken:   serviceToken,
		hub_base_URL:   baseURL,
		mgmnt_base_URL: mgmntURL,
		debug:          debug,
	}
	client.workspaceClients = make(map[string]*suprsend.Client)
	log.Debugf("New management client created with base URL: %s, mgmnt URL: %s and debug: %t", baseURL, mgmntURL, debug)
	return client
}

// NewClientWithUrlsAndTransport behaves like NewClientWithUrls but lets the
// caller inject a shared http.RoundTripper. The transport is applied to:
//   - the workspace key/secret lookup HTTP client (in GetWorkspaceKeyAndSecret)
//   - the resty client used for schema fetches (via resty.SetTransport)
//   - the suprsend-go workspace client (via suprsend.WithHTTPClient)
//
// Used by multi-tenant MCP servers (internal/utils.MgmntClientFor) to wrap
// an authExpiryTransport that calls mcpserver.MarkSessionDead on HTTP 401.
func NewClientWithUrlsAndTransport(serviceToken, baseURL, mgmntURL string, transport http.RoundTripper, debug bool) *SS_MgmntClient {
	c := NewClientWithUrls(serviceToken, baseURL, mgmntURL, debug)
	c.transport = transport
	return c
}

// httpClient returns an *http.Client honoring c.transport. Always sets a
// 10-second timeout — both the workspace key/secret lookup and the
// management API should respond well within that, and an unbounded client
// makes the auth path hangable.
func (c *SS_MgmntClient) httpClient() *http.Client {
	rt := c.transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &http.Client{Transport: rt, Timeout: 10 * time.Second}
}

// restyClient returns a configured resty.Client that honors c.transport
// (an authExpiryTransport, when set by a pkg/mcpserver embedder) and a 10s
// default timeout. ALL mgmnt-package HTTP construction MUST go through this
// method instead of calling client.NewHTTPClient() or resty.New() directly
// — otherwise the auth-expiry interceptor and the timeout are bypassed.
func (c *SS_MgmntClient) restyClient() *resty.Client {
	return client.NewHTTPClientWithOptions(client.Options{
		Transport: c.transport,
		Timeout:   10 * time.Second,
	})
}

// GetWorkspaceClient returns a cached suprsend workspace client. Uses a
// background context for the workspace key/secret lookup. Prefer
// GetWorkspaceClientCtx from any code path that already has a request context.
func (c *SS_MgmntClient) GetWorkspaceClient(workspace string) (*suprsend.Client, error) {
	return c.GetWorkspaceClientCtx(context.Background(), workspace)
}

// GetWorkspaceClientCtx is the context-aware variant of GetWorkspaceClient.
// The ctx is propagated to the workspace key/secret lookup; once the
// workspace client is cached it is reused for the lifetime of c.
func (c *SS_MgmntClient) GetWorkspaceClientCtx(ctx context.Context, workspace string) (*suprsend.Client, error) {
	// Store a hashmap of workspaces and their clients, if the client doesn't exist, create it
	if c.workspaceClients[workspace] == nil {
		key, secret, err := c.GetWorkspaceKeyAndSecret(ctx, workspace)
		if err != nil {
			return nil, err
		}
		opts := []suprsend.ClientOption{suprsend.WithBaseUrl(c.hub_base_URL), suprsend.WithDebug(c.debug)}
		if c.transport != nil {
			opts = append(opts, suprsend.WithHTTPClient(&http.Client{Transport: c.transport, Timeout: 30 * time.Second}))
		}
		client, err := suprsend.NewClient(key, secret, opts...)
		if err != nil {
			return nil, err
		}
		log.Debugf("New workspace client created for workspace: %s and debug: %t", workspace, c.debug)
		c.workspaceClients[workspace] = client
	}

	return c.workspaceClients[workspace], nil
}

// Function to get workspace key and secret for a given workspace name
func (c *SS_MgmntClient) GetWorkspaceKeyAndSecret(ctx context.Context, workspace string) (string, string, error) {
	// Look up the workspace key/secret via the hub API, passing the service
	// token as an Authorization header.

	// Use the shared client so the injected transport (and the 10s timeout) apply.
	client := c.httpClient()

	// Create a new GET request
	urlStr, err := url.JoinPath(c.hub_base_URL, "v1", workspace, "ws_key", "bridge", "/")
	if err != nil {
		log.Info("Error creating request: ", err)
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		log.Info("Error creating request: ", err)
		return "", "", err
	}
	req.Header.Set("Authorization", "ServiceToken "+c.serviceToken)

	// Send the request
	response, err := client.Do(req)
	if err != nil {
		log.Info("Error sending request: ", err)
		return "", "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Info("Error reading response body: ", err)
		return "", "", err
	}
	var workspaceDetails struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}
	err = json.Unmarshal(body, &workspaceDetails)
	if err != nil {
		return "", "", errors.New("failed to initialize suprsend workspace client")
	}
	if workspaceDetails.Key == "" || workspaceDetails.Secret == "" {
		return "", "", errors.New("failed to initialize suprsend workspace client")
	}
	return workspaceDetails.Key, workspaceDetails.Secret, nil
}
