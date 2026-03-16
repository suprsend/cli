package profiles

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var removeName string

var profileRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a profile",
	Long:  "Remove a profile from the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if removeName == "" {
			removeName = promptForProfileName()
			if removeName == "" {
				log.Error("No profile name provided")
				return nil
			}
		}

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

		if _, exists := cfg.Profiles[removeName]; !exists {
			log.Infof("Profile %q does not exist. Use the command 'suprsend profile list' to see all profiles.", removeName)
			return nil
		}

		delete(cfg.Profiles, removeName)
		log.Printf("Profile %q has been removed.", removeName)

		if cfg.ActiveProfile == removeName {
			if _, hasDefault := cfg.Profiles["default"]; hasDefault {
				cfg.ActiveProfile = "default"
				log.Warn("Active profile was removed. Falling back to 'default'.")
			} else {
				cfg.ActiveProfile = ""
				log.Warn("Active profile was removed. No active profile set.")
			}
		}

		if err := SaveConfig(cfg, path); err != nil {
			log.WithError(err).Error("Failed to save")
			return err
		}
		return nil
	},
}

func init() {
	profileRemoveCmd.Flags().StringVar(&removeName, "name", "", "Name of the profile to remove")
	ProfileCmd.AddCommand(profileRemoveCmd)
}
