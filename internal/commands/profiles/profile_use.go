package profiles

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

var useName string

var profileUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Set the active profile",
	Long:  "Set the active profile in the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("config")
		if err != nil {
			log.WithError(err).Error("Couldn't find the path")
			return clierr.Wrap(err, clierr.CodeUnknown, "")
		}

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load config")
			return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
		}

		if useName == "" {
			useName = promptForProfileToUse(cfg)
			if useName == "" {
				return clierr.New("required flag missing (--name), cannot prompt in non-interactive mode", clierr.CodeInvalidUsage)
			}
		}

		if _, exists := cfg.Profiles[useName]; !exists {
			log.Infof("Profile %q does not exist. Use the command 'suprsend profiles list' to see all profiles.", useName)
			return clierr.New(fmt.Sprintf("profile %q does not exist", useName), clierr.CodeUnknown)
		}

		cfg.ActiveProfile = useName

		if err := config.SaveProfileConfig(cfg, path); err != nil {
			log.WithError(err).Error("Failed to save config")
			return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
		}

		log.Infof("Active profile set to %q.", useName)
		return nil
	},
}

func init() {
	profileUseCmd.Flags().StringVarP(&useName, "name", "", "", "Profile name to set as active")
	ProfileCmd.AddCommand(profileUseCmd)
}

func promptForProfileToUse(cfg *config.ProfileConfig) string {
	if !utils.IsInputInteractive() {
		return ""
	}
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
