package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

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
	// Set reports whether a flag value was passed explicitly rather than defaulted.
	Set bool `yaml:"-"`
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
	// Profile is the active profile's name, empty when no config file is used.
	Profile ConfigString
	// Workspace is set by ResolveWorkspace for commands that take --workspace.
	Workspace ConfigString
	// flag only configs (not resolved from env/profile)
	CfgFile    ConfigString
	OutputType ConfigString
	Verbosity  ConfigString
	Quiet      ConfigBool
	Debug      ConfigBool

	profileWorkspace string
}

const (
	DefaultBaseUrl   = "https://hub.suprsend.com/"
	DefaultMgmntUrl  = "https://management-api.suprsend.com/"
	DefaultWorkspace = "staging"
)

// Cfg is the global configuration instance.
var Cfg = &Config{}

// FlagValues holds the raw values parsed from CLI flags before any resolution.
type FlagValues struct {
	CfgFile      string
	OutputType   string
	Verbosity    string
	VerbositySet bool // -v was passed explicitly
	ServiceToken string
	NoColor      bool
	Quiet        bool
}

func GetResolvedServiceToken(flagToken string, activeProfile Profile) (ConfigString, error) {
	if flagToken != "" {
		if os.Getenv("SUPRSEND_SERVICE_TOKEN") != "" {
			log.Debug("Using service token from command line flag (overrides SUPRSEND_SERVICE_TOKEN)")
		} else {
			log.Debug("Using service token from command line flag")
		}
		return ConfigString{Value: flagToken, Source: ConfigSourceFlag}, nil
	}
	if envToken := os.Getenv("SUPRSEND_SERVICE_TOKEN"); envToken != "" {
		log.Debug("Using service token from environment variable")
		return ConfigString{Value: envToken, Source: ConfigSourceEnv}, nil
	}
	if activeProfile.ServiceToken.Value != "" {
		log.Debug("Using service token from config file profile")
		return ConfigString{Value: activeProfile.ServiceToken.Value, Source: ConfigSourceProfile}, nil
	}
	return ConfigString{}, clierr.New("no service token found in command line, environment, or config file", clierr.CodeAuthMissingToken).
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
	debugVal, err := strconv.ParseBool(v)
	if err != nil && v != "" {
		log.Warnf("invalid DEBUG=%q: expected true/false/1/0; treating as false", v)
	}
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
// Priority varies by field:
//   - ServiceToken: flag > env > profile (no default — errors if unset)
//   - Workspace: flag > env > profile > default, resolved per command by ResolveWorkspace
//   - BaseUrl, MgmntUrl: env > profile > default
//   - NoColorOutput: env > flag > default
//   - Debug, ProxyURL: env only
//   - OutputType, Verbosity, Quiet, CfgFile: flag only
func (c *Config) Resolve(flags FlagValues) error {
	c.CfgFile = ConfigString{Value: flags.CfgFile, Source: ConfigSourceFlag}
	if flags.CfgFile != "" {
		if _, err := os.Stat(flags.CfgFile); err != nil {
			return clierr.Wrap(err, clierr.CodeConfigInvalid, fmt.Sprintf("cannot read config file %s", flags.CfgFile))
		}
	}
	var activeProfile Profile
	if configPath := GetConfigFilePath(); configPath != "" {
		if cfg, err := LoadProfileConfig(configPath); err == nil {
			activeProfile = cfg.Profiles[cfg.ActiveProfile]
			if cfg.ActiveProfile != "" {
				c.Profile = ConfigString{Value: cfg.ActiveProfile, Source: ConfigSourceProfile}
			}
			log.Debug("Using config file:", configPath)
		} else if flags.CfgFile != "" {
			return clierr.Wrap(err, clierr.CodeConfigInvalid, fmt.Sprintf("cannot parse config file %s", configPath))
		} else {
			log.Debugf("failed to load config file %s: %v", configPath, err)
		}
	}

	c.profileWorkspace = activeProfile.Workspace.Value
	c.OutputType = ConfigString{Value: flags.OutputType, Source: ConfigSourceFlag}
	c.Verbosity = ConfigString{Value: flags.Verbosity, Source: ConfigSourceFlag, Set: flags.VerbositySet}
	c.Quiet = ConfigBool{Value: flags.Quiet, Source: ConfigSourceFlag}
	c.Debug = GetResolvedDebug()

	c.NoColorOutput = GetResolvedNoColor(flags.NoColor)
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

// WorkspaceFlagHelp is the help text for every --workspace flag.
const WorkspaceFlagHelp = "Workspace slug, e.g. staging or production (default: $SUPRSEND_WORKSPACE, else the profile's workspace, else staging)"

var workspacePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ResolveWorkspace picks the workspace for a command: the --workspace flag
// when it was passed, else SUPRSEND_WORKSPACE, else the active profile's
// workspace, else DefaultWorkspace. Surrounding whitespace is trimmed (a
// shell quoting slip like "-w staging" as one argument would otherwise send
// " staging" to the API, which answers only "resource not found"), and a
// value that can't be a workspace slug is an error. The result is stored in
// Cfg.Workspace.
func (c *Config) ResolveWorkspace(flagValue string, flagSet bool) (ConfigString, error) {
	ws := ConfigString{Value: DefaultWorkspace, Source: ConfigSourceDefault}
	switch {
	case flagSet:
		ws = ConfigString{Value: flagValue, Source: ConfigSourceFlag}
	case os.Getenv("SUPRSEND_WORKSPACE") != "":
		ws = ConfigString{Value: os.Getenv("SUPRSEND_WORKSPACE"), Source: ConfigSourceEnv}
	case c.profileWorkspace != "":
		ws = ConfigString{Value: c.profileWorkspace, Source: ConfigSourceProfile}
	}
	value, err := NormalizeWorkspace(ws.Value, "from "+string(ws.Source))
	if err != nil {
		return ConfigString{}, err
	}
	ws.Value = value
	c.Workspace = ws
	return ws, nil
}

// NormalizeWorkspace trims value and checks that it looks like a workspace
// slug. from says where the value came from, for the error message.
func NormalizeWorkspace(value, from string) (string, error) {
	if trimmed := strings.TrimSpace(value); trimmed != value {
		log.Warnf("workspace %q has surrounding whitespace; using %q", value, trimmed)
		value = trimmed
	}
	if !workspacePattern.MatchString(value) {
		return "", clierr.New(fmt.Sprintf("invalid workspace %q (%s): expected a workspace slug such as staging or production", value, from), clierr.CodeInvalidUsage).
			WithHint("run `suprsend workspace list` to see the workspace slugs")
	}
	return value, nil
}
