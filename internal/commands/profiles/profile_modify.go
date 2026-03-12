package profiles

import (
	"fmt"

	"github.com/sabouaram/cobra_ui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	modifyName         string
	modifyBaseUrl      string
	modifyMgmntUrl     string
	modifyServiceToken string
)

var profilesModifyCmd = &cobra.Command{
	Use:   "modify",
	Short: "Modify a profile",
	Long:  "Modify a profile in the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load or create config")
			return err
		}
		if modifyName != "" {
			if _, exists := cfg.Profiles[modifyName]; !exists {
				return fmt.Errorf("profile %q does not exist. Use the command 'suprsend profile list' to see all profiles", modifyName)
			}
		}

		if modifyName != "" && modifyServiceToken != "" {
			selectedProfile := cfg.Profiles[modifyName]

			if modifyBaseUrl != "" {
				selectedProfile.BaseUrl = modifyBaseUrl
			}
			if modifyMgmntUrl != "" {
				selectedProfile.MgmntUrl = modifyMgmntUrl
			}
			selectedProfile.ServiceToken = modifyServiceToken

			cfg.Profiles[modifyName] = selectedProfile

			err := SaveConfig(cfg, path)
			if err != nil {
				log.WithError(err).Error("Failed to save config")
				return err
			}

			log.Infof("Profile %s modified successfully", modifyName)
		} else {
			runModifyInteractive(cfg, path)
		}
		return nil
	},
}

func init() {
	profilesModifyCmd.Flags().StringVar(&modifyName, "name", "", "Name of the profile to modify")
	profilesModifyCmd.Flags().StringVar(&modifyBaseUrl, "base-url", "", "Base URL (default: https://hub.suprsend.com/)")
	profilesModifyCmd.Flags().StringVar(&modifyMgmntUrl, "mgmnt-url", "", "Management URL (default: https://management-api.suprsend.com/)")
	profilesModifyCmd.Flags().StringVar(&modifyServiceToken, "service-token", "", "Service Token")
	ProfileCmd.AddCommand(profilesModifyCmd)
}

func runModifyInteractive(cfg *Config, path string) {
	ui := cobra_ui.New()

	var profileNames []string
	for name := range cfg.Profiles {
		profileNames = append(profileNames, name)
	}

	if len(profileNames) == 0 {
		log.Info("No profiles found. Use the command 'suprsend profiles add' to add a profile.")
		return
	}

	if modifyName == "" {
		ui.SetQuestions([]cobra_ui.Question{
			{
				Text: "Select a profile to modify: ",
				Handler: func(s string) error {
					s = cleanInput(s)
					modifyName = s
					return nil
				},
				Options: profileNames,
			},
		})

		ui.RunInteractiveUI()
	}

	selectedProfile, exists := cfg.Profiles[modifyName]
	if !exists {
		log.Infof("Profile %q does not exist. Use the command 'suprsend profile list' to see all profiles.", modifyName)
		return
	}

	ui2 := cobra_ui.New()

	if modifyBaseUrl == "" {
		modifyBaseUrl = selectedProfile.BaseUrl
		if modifyBaseUrl == "" {
			modifyBaseUrl = "https://hub.suprsend.com/"
		}
	}
	if modifyMgmntUrl == "" {
		modifyMgmntUrl = selectedProfile.MgmntUrl
		if modifyMgmntUrl == "" {
			modifyMgmntUrl = "https://management-api.suprsend.com/"
		}
	}

	var questions []cobra_ui.Question

	if modifyServiceToken == "" {
		currentToken := selectedProfile.ServiceToken
		maskedToken := MaskServiceToken(currentToken)
		questions = append(questions, cobra_ui.Question{
			Text: fmt.Sprintf("Service Token (current: %s, press Enter to keep): ", maskedToken),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s != "" {
					modifyServiceToken = s
				} else {
					modifyServiceToken = selectedProfile.ServiceToken
				}
				return nil
			},
		})
	}

	if len(questions) > 0 {
		ui2.SetQuestions(questions)
		ui2.RunInteractiveUI()
	}

	if modifyName == "" {
		log.Error("Profile name is required")
		return
	}
	if modifyServiceToken == "" {
		log.Error("Service token is required")
		return
	}

	updatedProfile := Profile{
		BaseUrl:      modifyBaseUrl,
		MgmntUrl:     modifyMgmntUrl,
		ServiceToken: modifyServiceToken,
	}

	cfg.Profiles[modifyName] = updatedProfile

	err := SaveConfig(cfg, path)
	if err != nil {
		log.WithError(err).Error("Failed to save config")
		return
	}

	log.Infof("Profile %s modified successfully", modifyName)
}
