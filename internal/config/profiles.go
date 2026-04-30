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

type ConfigSource string

const (
	ConfigSourceEnv     ConfigSource = "env"
	ConfigSourceFlag    ConfigSource = "flag"
	ConfigSourceProfile ConfigSource = "profile"
	ConfigSourceDefault ConfigSource = "default"
)

type ConfigString struct {
	Value  string
	Source ConfigSource
}

func (p ConfigString) MarshalYAML() (interface{}, error) {
	return p.Value, nil
}

func (p *ConfigString) UnmarshalYAML(value *yaml.Node) error {
	p.Value = value.Value
	return nil
}

func (p ConfigString) String() string {
	return p.Value
}

type Profile struct {
	BaseUrl      ConfigString `yaml:"base_url"`
	MgmntUrl     ConfigString `yaml:"mgmnt_url"`
	ServiceToken ConfigString `yaml:"service_token"`
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

func GetResolvedServiceToken(flagToken string, activeProfile Profile) (ConfigString, error) {
	if envToken := os.Getenv("SUPRSEND_SERVICE_TOKEN"); envToken != "" {
		log.Debug("Using service token from environment variable")
		return ConfigString{Value: envToken, Source: ConfigSourceEnv}, nil
	}
	if flagToken != "" {
		log.Debug("Using service token from command line flag")
		return ConfigString{Value: flagToken, Source: ConfigSourceFlag}, nil
	}
	if activeProfile.ServiceToken.Value != "" {
		log.Debug("Using service token from config file profile")
		return ConfigString{Value: activeProfile.ServiceToken.Value, Source: ConfigSourceProfile}, nil
	}
	return ConfigString{}, clierr.New("no service token found in environment, command line, or config file", clierr.CodeAuthMissingToken).
		WithHint("set SUPRSEND_SERVICE_TOKEN or run `suprsend profile add`")
}

func GetResolvedBaseUrl(activeProfile Profile) ConfigString {
	if envUrl := os.Getenv("SUPRSEND_BASE_URL"); envUrl != "" {
		return ConfigString{Value: envUrl, Source: ConfigSourceEnv}
	}
	if activeProfile.BaseUrl.Value != "" {
		return ConfigString{Value: activeProfile.BaseUrl.Value, Source: ConfigSourceProfile}
	}
	return ConfigString{Value: DefaultBaseUrl, Source: ConfigSourceDefault}
}

func GetResolvedMgmntUrl(activeProfile Profile) ConfigString {
	if envUrl := os.Getenv("SUPRSEND_MGMNT_URL"); envUrl != "" {
		return ConfigString{Value: envUrl, Source: ConfigSourceEnv}
	}
	if activeProfile.MgmntUrl.Value != "" {
		return ConfigString{Value: activeProfile.MgmntUrl.Value, Source: ConfigSourceProfile}
	}
	return ConfigString{Value: DefaultMgmntUrl, Source: ConfigSourceDefault}
}

// FlagValues holds the raw values parsed from CLI flags before any resolution.
type FlagValues struct {
	Workspace    string
	CfgFile      string
	OutputType   string
	Verbosity    string
	ServiceToken string
	NoColor      bool
	Quiet        bool
}

// Resolve populates c with all flag-derived and env-var / profile-resolved values.
// Priority: env var > CLI flag > active config-file profile > hardcoded default.
func (c *Config) Resolve(flags FlagValues) error {
	c.Workspace = flags.Workspace
	c.CfgFile = flags.CfgFile
	c.OutputType = flags.OutputType
	c.Verbosity = flags.Verbosity
	c.NoColorOutput = flags.NoColor
	c.Quiet = flags.Quiet

	var activeProfile Profile
	if configPath := GetConfigFilePath(); configPath != "" {
		if cfg, err := LoadProfileConfig(configPath); err == nil {
			activeProfile = cfg.Profiles[cfg.ActiveProfile]
		}
	}

	token, err := GetResolvedServiceToken(flags.ServiceToken, activeProfile)
	if err != nil {
		return err
	}
	c.ServiceToken = token
	c.BaseUrl = GetResolvedBaseUrl(activeProfile)
	c.MgmntUrl = GetResolvedMgmntUrl(activeProfile)
	if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		c.ProxyURL = ConfigString{Value: proxyURL, Source: ConfigSourceEnv}
	}
	return nil
}
