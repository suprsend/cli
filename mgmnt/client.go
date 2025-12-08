package mgmnt

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	suprsend "github.com/suprsend/suprsend-go"
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

// Add method to client to get WorkspaceClient
func (c *SS_MgmntClient) GetWorkspaceClient(workspace string) (*suprsend.Client, error) {
	// Store a hashmap of workspaces and their clients, if the client doesn't exist, create it
	if c.workspaceClients[workspace] == nil {
		key, secret, err := c.GetWorkspaceKeyAndSecret(workspace)
		if err != nil {
			return nil, err
		}
		client, err := suprsend.NewClient(key, secret, suprsend.WithBaseUrl(c.hub_base_URL), suprsend.WithDebug(c.debug))
		if err != nil {
			return nil, err
		}
		log.Debugf("New workspace client created for workspace: %s and debug: %t", workspace, c.debug)
		c.workspaceClients[workspace] = client
	}

	return c.workspaceClients[workspace], nil
}

// Function to get workspace key and secret for a given workspace name
func (c *SS_MgmntClient) GetWorkspaceKeyAndSecret(workspace string) (string, string, error) {
	// Make a GET request to bridge API to get workspace key and secret, pass in service token as header

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new GET request
	requestURL, err := url.JoinPath(c.hub_base_URL, "v1", workspace, "ws_key", "bridge")
	if err != nil {
		log.Info("Error joining URL path: ", err)
		return "", "", err
	}
	req, err := http.NewRequest("GET", requestURL, nil)
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
