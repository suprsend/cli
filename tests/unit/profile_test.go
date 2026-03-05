package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suprsend/cli/internal/commands/profiles"
)

func TestMaskServiceToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", "not set"},
		{"short (4 chars)", "abcd", "****"},
		{"exactly 8 chars", "abcdefgh", "****"},
		{"longer than 8", "abcdefghijkl", "abcd****ijkl"},
		{"9 chars", "abcdefghi", "abcd****fghi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profiles.MaskServiceToken(tt.input)
			if got != tt.want {
				t.Errorf("MaskServiceToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &profiles.Config{
		ActiveProfile: "default",
		Profiles: map[string]profiles.Profile{
			"default": {
				BaseUrl:      "https://hub.suprsend.com/",
				ServiceToken: "test-token",
			},
		},
	}

	err := profiles.SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	content := string(data)
	if content == "" {
		t.Error("config file should not be empty")
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `active_profile: default
profiles:
    default:
        base_url: https://hub.suprsend.com/
        mgmnt_url: https://management-api.suprsend.com/
        service_token: my-token
`
	os.WriteFile(configPath, []byte(yamlContent), 0644)

	cfg, err := profiles.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want %q", cfg.ActiveProfile, "default")
	}
	if len(cfg.Profiles) != 1 {
		t.Errorf("Profiles count = %d, want 1", len(cfg.Profiles))
	}
	prof := cfg.Profiles["default"]
	if prof.ServiceToken != "my-token" {
		t.Errorf("ServiceToken = %q, want %q", prof.ServiceToken, "my-token")
	}
	if prof.BaseUrl != "https://hub.suprsend.com/" {
		t.Errorf("BaseUrl = %q, want %q", prof.BaseUrl, "https://hub.suprsend.com/")
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	_, err := profiles.LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	original := &profiles.Config{
		ActiveProfile: "staging",
		Profiles: map[string]profiles.Profile{
			"staging": {
				BaseUrl:      "https://staging.suprsend.com/",
				MgmntUrl:     "https://staging-mgmnt.suprsend.com/",
				ServiceToken: "staging-token-12345",
			},
			"prod": {
				BaseUrl:      "https://hub.suprsend.com/",
				ServiceToken: "prod-token-67890",
			},
		},
	}

	if err := profiles.SaveConfig(original, configPath); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := profiles.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.ActiveProfile != original.ActiveProfile {
		t.Errorf("ActiveProfile = %q, want %q", loaded.ActiveProfile, original.ActiveProfile)
	}
	if len(loaded.Profiles) != len(original.Profiles) {
		t.Errorf("Profiles count = %d, want %d", len(loaded.Profiles), len(original.Profiles))
	}
	for name, orig := range original.Profiles {
		loaded := loaded.Profiles[name]
		if loaded.BaseUrl != orig.BaseUrl {
			t.Errorf("Profile %q BaseUrl = %q, want %q", name, loaded.BaseUrl, orig.BaseUrl)
		}
		if loaded.ServiceToken != orig.ServiceToken {
			t.Errorf("Profile %q ServiceToken = %q, want %q", name, loaded.ServiceToken, orig.ServiceToken)
		}
	}
}
