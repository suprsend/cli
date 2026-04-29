package profiles

import (
	"bufio"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	BaseUrl      string `yaml:"base_url"`
	MgmntUrl     string `yaml:"mgmnt_url"`
	ServiceToken string `yaml:"service_token"`
}

type Config struct {
	ActiveProfile string             `yaml:"active_profile"`
	Profiles      map[string]Profile `yaml:"profiles"`
}

type ProfileListItem struct {
	Name         string `json:"name" yaml:"name"`
	Active       string `json:"active" yaml:"active"`
	BaseUrl      string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	MgmntUrl     string `json:"mgmnt_url,omitempty" yaml:"mgmnt_url,omitempty"`
	ServiceToken string `json:"service_token,omitempty" yaml:"service_token,omitempty"`
}

type SimpleProfileListItem struct {
	Name   string `json:"name" yaml:"name"`
	Active string `json:"active" yaml:"active"`
}

func cleanInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "[")
	input = strings.TrimSuffix(input, "]")
	return input
}

// Public-cloud defaults; used when a profile doesn't override these.
const (
	DefaultBaseUrl  = "https://hub.suprsend.com/"
	DefaultMgmntUrl = "https://management-api.suprsend.com/"
)

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

func EnsureConfig(path string) (*Config, string, error) {
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

		defaultCfg := &Config{
			ActiveProfile: "",
			Profiles:      make(map[string]Profile),
		}
		if err := SaveConfig(defaultCfg, configPath); err != nil {
			log.WithError(err).Error("Failed to create default config")
			return nil, configPath, err
		}
		log.Infof("Created default config at %s", configPath)
		return defaultCfg, configPath, nil
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.WithError(err).Error("Failed to load config file")
		return nil, configPath, err
	}
	return cfg, configPath, nil
}

func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetConfigFilePath() string {
	if config.Cfg.CfgFile != "" {
		return config.Cfg.CfgFile
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.WithError(err).Error("Could not get user home directory")
		return ""
	}
	return filepath.Join(homeDir, ".suprsend.yaml")
}

func GetResolvedBaseUrl() string {
	// ENV Variable
	if envUrl := os.Getenv("SUPRSEND_BASE_URL"); envUrl != "" {
		return envUrl
	}

	// get the value from the active profile
	configPath := GetConfigFilePath()
	if configPath == "" {
		return DefaultBaseUrl
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return DefaultBaseUrl
	}

	activeProfile := cfg.Profiles[cfg.ActiveProfile]
	if activeProfile.BaseUrl != "" {
		return activeProfile.BaseUrl
	}

	// Default value
	return DefaultBaseUrl
}

func GetResolvedMgmntUrl() string {
	// ENV Variable
	if envUrl := os.Getenv("SUPRSEND_MGMNT_URL"); envUrl != "" {
		return envUrl
	}

	// get the value from the active profile
	configPath := GetConfigFilePath()
	if configPath == "" {
		return DefaultMgmntUrl
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return DefaultMgmntUrl
	}

	activeProfile := cfg.Profiles[cfg.ActiveProfile]
	if activeProfile.MgmntUrl != "" {
		return activeProfile.MgmntUrl
	}

	// Default value
	return DefaultMgmntUrl
}

// MaskServiceToken masks a service token showing only first 4 and last 4 characters
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
