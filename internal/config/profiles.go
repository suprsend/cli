package config

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"gopkg.in/yaml.v3"
)

const (
	DefaultBaseUrl  = "https://hub.suprsend.com/"
	DefaultMgmntUrl = "https://management-api.suprsend.com/"
)

type Profile struct {
	BaseUrl      string `yaml:"base_url"`
	MgmntUrl     string `yaml:"mgmnt_url"`
	ServiceToken string `yaml:"service_token"`
}

type ProfileConfig struct {
	ActiveProfile string             `yaml:"active_profile"`
	Profiles      map[string]Profile `yaml:"profiles"`
}

func GetConfigFilePath() string {
	if Cfg.CfgFile != "" {
		return Cfg.CfgFile
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.WithError(err).Error("Could not get user home directory")
		return ""
	}
	return filepath.Join(homeDir, ".suprsend.yaml")
}

func LoadProfileConfig(path string) (*ProfileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ProfileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveProfileConfig(cfg *ProfileConfig, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func GetResolvedServiceToken(flagToken string, activeProfile Profile) string {
	if envToken := os.Getenv("SUPRSEND_SERVICE_TOKEN"); envToken != "" {
		log.Debug("Using service token from environment variable")
		return envToken
	}
	if flagToken != "" {
		log.Debug("Using service token from command line flag")
		return flagToken
	}
	if activeProfile.ServiceToken != "" {
		log.Debug("Using service token from config file profile")
		return activeProfile.ServiceToken
	}
	return ""
}

func GetResolvedBaseUrl(activeProfile Profile) string {
	if envUrl := os.Getenv("SUPRSEND_BASE_URL"); envUrl != "" {
		return envUrl
	}
	if activeProfile.BaseUrl != "" {
		return activeProfile.BaseUrl
	}
	return DefaultBaseUrl
}

func GetResolvedMgmntUrl(activeProfile Profile) string {
	if envUrl := os.Getenv("SUPRSEND_MGMNT_URL"); envUrl != "" {
		return envUrl
	}
	if activeProfile.MgmntUrl != "" {
		return activeProfile.MgmntUrl
	}
	return DefaultMgmntUrl
}

// Resolve populates c with all env-var / flag / profile-resolved values.
// Priority: env var > CLI flag > active config-file profile > hardcoded default.
// The profile config file is loaded once and passed to each resolver.
func (c *Config) Resolve(flagToken string) error {
	var activeProfile Profile
	if configPath := GetConfigFilePath(); configPath != "" {
		if cfg, err := LoadProfileConfig(configPath); err == nil {
			activeProfile = cfg.Profiles[cfg.ActiveProfile]
		}
	}

	c.ServiceToken = GetResolvedServiceToken(flagToken, activeProfile)
	if c.ServiceToken == "" {
		return clierr.New("no service token found in environment, command line, or config file", clierr.CodeAuthMissingToken).
			WithHint("set SUPRSEND_SERVICE_TOKEN or run `suprsend profile add`")
	}
	c.BaseUrl = GetResolvedBaseUrl(activeProfile)
	c.MgmntUrl = GetResolvedMgmntUrl(activeProfile)
	c.ProxyURL = os.Getenv("HTTP_PROXY")
	return nil
}
