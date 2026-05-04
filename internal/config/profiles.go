package config

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

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
	if Cfg.CfgFile.Value != "" {
		return Cfg.CfgFile.Value
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
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
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

