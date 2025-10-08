package profiles

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	useName string
)

var profileUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Set the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("config")
		if err != nil {
			log.WithError(err).Error("Couldn't find the path")
			return err
		}

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load config")
			return err
		}

		if useName == "" {
			useName = promptForProfileToUse(cfg)
			if useName == "" {
				log.Error("No profile name provided")
				return err
			}
		}

		if _, exists := cfg.Profiles[useName]; !exists {
			log.Infof("Profile %q does not exist. Use the command 'suprsend profiles list' to see all profiles.", useName)
			return err
		}

		cfg.ActiveProfile = useName

		if err := SaveConfig(cfg, path); err != nil {
			log.WithError(err).Error("Failed to save config")
			return err
		}

		log.Infof("Active profile set to %q.", useName)
		return nil
	},
}

func init() {
	profileUseCmd.Flags().StringVarP(&useName, "name", "", "", "Profile name to set as active")
	ProfileCmd.AddCommand(profileUseCmd)
}

func promptForProfileToUse(cfg *Config) string {
	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles found. Create a profile first with 'suprsend profiles add'")
		return ""
	}

	fmt.Println("Available profiles:")
	for name := range cfg.Profiles {
		active := ""
		if name == cfg.ActiveProfile {
			active = " (current)"
		}
		fmt.Printf("  - %s%s\n", name, active)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter profile name to use: ")
	name, _ := reader.ReadString('\n')
	return strings.TrimSpace(name)
}
