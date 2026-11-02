package profiles

import (
	"bufio"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

type ProfileListItem struct {
	Name         string `json:"name" yaml:"name"`
	Active       string `json:"active" yaml:"active"`
	BaseUrl      string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	MgmntUrl     string `json:"mgmnt_url,omitempty" yaml:"mgmnt_url,omitempty"`
	Workspace    string `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	ServiceToken string `json:"service_token,omitempty" yaml:"service_token,omitempty"`
}

type SimpleProfileListItem struct {
	Name   string `json:"name" yaml:"name"`
	Active string `json:"active" yaml:"active"`
}

// question is one interactive prompt. Handler validates and stores the
// answer; an error is printed and the question asked again. With Options, the
// answer may be an option's number or its text.
type question struct {
	Text    string
	Options []string
	Handler func(string) error
}

// ask runs questions in order on stdin. It reads plain lines rather than
// using a terminal UI library, so it never sends terminal queries or escape
// sequences.
func ask(questions []question) error {
	reader := bufio.NewReader(os.Stdin)
	for _, q := range questions {
		for {
			for i, opt := range q.Options {
				fmt.Printf("  %d) %s\n", i+1, opt)
			}
			fmt.Print(q.Text)
			line, err := reader.ReadString('\n')
			if err != nil && line == "" {
				fmt.Println()
				return fmt.Errorf("no answer to %q: %w", strings.TrimSpace(q.Text), err)
			}
			answer := strings.TrimSpace(line)
			if n, convErr := strconv.Atoi(answer); convErr == nil && n >= 1 && n <= len(q.Options) {
				answer = q.Options[n-1]
			}
			if err := q.Handler(answer); err != nil {
				fmt.Printf("  %v\n", err)
				continue
			}
			break
		}
	}
	return nil
}

func cleanInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "[")
	input = strings.TrimSuffix(input, "]")
	return input
}

// validateAndNormalizeUrl parses, validates and normalizes a profile URL.
// Rules:
//   - leading / trailing whitespace is stripped
//   - scheme must be http or https
//   - host must be non-empty
//   - query strings and fragments are rejected (a SuprSend BYOC URL is a
//     base URL — `?foo=bar` or `#anchor` is almost always a typo)
//   - the path is normalized to a trailing slash so the rest of the
//     codebase can append routes without double-slash hazards
//
// Returns the normalized URL string. The caller should always store the
// returned value rather than the user-supplied input.
func validateAndNormalizeUrl(urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https")
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must include a host")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", fmt.Errorf("URL must not include a query string or fragment")
	}
	if !strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path += "/"
	}
	return parsedURL.String(), nil
}

func promptForProfileName() string {
	if !utils.IsInputInteractive() {
		return ""
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter profile name to remove: ")
	name, _ := reader.ReadString('\n')
	return strings.TrimSpace(name)
}

// EnsureConfig loads the profile config file, creating a default one if it
// doesn't exist and the terminal is interactive.
func EnsureConfig(path string) (*config.ProfileConfig, string, error) {
	var configPath string
	if path != "" {
		configPath = path
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.WithError(err).Error("Could not get user home directory")
			return nil, "", err
		}
		configPath = filepath.Join(homeDir, ".suprsend.yaml")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if !utils.IsInputInteractive() {
			return nil, configPath, clierr.New("config file not found and cannot prompt in non-interactive mode", clierr.CodeInvalidUsage)
		}
		log.Warnf("No config found at %s", configPath)
		log.Info("Would you like to create a default config? (Y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		response := scanner.Text()
		if response != "y" && response != "Y" {
			log.Error("Config file is required to proceed.")
			return nil, configPath, err
		}

		defaultCfg := &config.ProfileConfig{
			ActiveProfile: "",
			Profiles:      make(map[string]config.Profile),
		}
		if err := config.SaveProfileConfig(defaultCfg, configPath); err != nil {
			log.WithError(err).Error("Failed to create default config")
			return nil, configPath, err
		}
		log.Infof("Created default config at %s", configPath)
		return defaultCfg, configPath, nil
	}

	cfg, err := config.LoadProfileConfig(configPath)
	if err != nil {
		log.WithError(err).Error("Failed to load config file")
		return nil, configPath, err
	}
	return cfg, configPath, nil
}

// MaskServiceToken shows only first 4 and last 4 characters of a token.
func MaskServiceToken(token string) string {
	if token == "" {
		return "not set"
	}
	length := len(token)
	if length <= 8 {
		return "****"
	}
	maxCut := int(math.Min(4, float64(length)))
	return token[:maxCut] + "****" + token[length-4:]
}

// normalizeProfileWorkspace validates a --workspace value for a profile;
// empty means "no default workspace".
func normalizeProfileWorkspace(ws string) (string, error) {
	if strings.TrimSpace(ws) == "" {
		return "", nil
	}
	return config.NormalizeWorkspace(ws, "--workspace")
}
