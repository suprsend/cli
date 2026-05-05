package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/suprsend/cli/internal/clierr"
	"gopkg.in/yaml.v3"
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

type ConfigBool struct {
	Value    bool
	RawValue string
	Source   ConfigSource
}

func (p ConfigBool) MarshalYAML() (interface{}, error) {
	return p.Value, nil
}

func (p *ConfigBool) UnmarshalYAML(value *yaml.Node) error {
	return value.Decode(&p.Value)
}

func (p ConfigBool) Bool() bool {
	return p.Value
}

// Config holds the application's configuration.
type Config struct {
	ServiceToken  ConfigString
	NoColorOutput ConfigBool
	BaseUrl       ConfigString
	MgmntUrl      ConfigString
	ProxyURL      ConfigString
	// flag only configs (not resolved from env/profile)
	CfgFile    ConfigString
	Workspace  ConfigString
	OutputType ConfigString
	Verbosity  ConfigString
	Quiet      ConfigBool
	Debug      ConfigBool
}

const (
	DefaultBaseUrl  = "https://hub.suprsend.com/"
	DefaultMgmntUrl = "https://management-api.suprsend.com/"
)

// Cfg is the global configuration instance.
var Cfg = &Config{}

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

func GetResolvedDebug() ConfigBool {
	v := os.Getenv("DEBUG")
	debugVal, _ := strconv.ParseBool(v)
	return ConfigBool{Value: debugVal, RawValue: v, Source: ConfigSourceEnv}
}

func GetResolvedNoColor(flagNoColor bool) ConfigBool {
	if v := os.Getenv("NO_COLOR"); v != "" {
		return ConfigBool{Value: true, RawValue: v, Source: ConfigSourceEnv}
	}
	if flagNoColor {
		return ConfigBool{Value: true, Source: ConfigSourceFlag}
	}
	return ConfigBool{Value: false, Source: ConfigSourceDefault}
}

// Resolve populates c with all flag-derived and env-var / profile-resolved values.
// Priority: env var > CLI flag > active config-file profile > hardcoded default.
func (c *Config) Resolve(flags FlagValues) error {
	c.CfgFile = ConfigString{Value: flags.CfgFile, Source: ConfigSourceFlag}
	if flags.CfgFile != "" {
		if _, err := os.Stat(flags.CfgFile); err != nil {
			return fmt.Errorf("cannot read config file %s: %w", flags.CfgFile, err)
		}
	}
	var activeProfile Profile
	if configPath := GetConfigFilePath(); configPath != "" {
		if cfg, err := LoadProfileConfig(configPath); err == nil {
			activeProfile = cfg.Profiles[cfg.ActiveProfile]
			log.Debug("Using config file:", configPath)
		} else if flags.CfgFile != "" {
			return clierr.Wrap(err, clierr.CodeConfigInvalid, fmt.Sprintf("cannot parse config file %s", configPath))
		} else {
			log.Debugf("failed to load config file %s: %v", configPath, err)
		}
	}

	c.Workspace = ConfigString{Value: flags.Workspace, Source: ConfigSourceFlag}
	c.OutputType = ConfigString{Value: flags.OutputType, Source: ConfigSourceFlag}
	c.Verbosity = ConfigString{Value: flags.Verbosity, Source: ConfigSourceFlag}
	c.Quiet = ConfigBool{Value: flags.Quiet, Source: ConfigSourceFlag}
	c.Debug = GetResolvedDebug()

	c.NoColorOutput = GetResolvedNoColor(flags.NoColor)
	if c.NoColorOutput.Value {
		color.NoColor = true
	}
	c.BaseUrl = GetResolvedBaseUrl(activeProfile)
	c.MgmntUrl = GetResolvedMgmntUrl(activeProfile)

	if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		c.ProxyURL = ConfigString{Value: proxyURL, Source: ConfigSourceEnv}
	}

	token, err := GetResolvedServiceToken(flags.ServiceToken, activeProfile)
	if err != nil {
		return err
	}
	c.ServiceToken = token

	return nil
}
