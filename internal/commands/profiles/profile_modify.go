package profiles

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

var (
	modifyName         string
	modifyBaseUrl      string
	modifyMgmntUrl     string
	modifyServiceToken string
	modifyWorkspace    string
)

var profilesModifyCmd = &cobra.Command{
	Use:   "modify",
	Short: "Modify a profile",
	Long:  "Modify a profile in the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
		}
		workspaceSet := cmd.Flags().Changed("workspace")
		if modifyWorkspace, err = normalizeProfileWorkspace(modifyWorkspace); err != nil {
			return err
		}
		if modifyName != "" {
			if _, exists := cfg.Profiles[modifyName]; !exists {
				return clierr.New(fmt.Sprintf("profile %q does not exist. Use the command 'suprsend profile list' to see all profiles", modifyName), clierr.CodeInvalidUsage)
			}
		}

		if modifyName != "" && (modifyServiceToken != "" || modifyBaseUrl != "" || modifyMgmntUrl != "" || workspaceSet) {
			selectedProfile := cfg.Profiles[modifyName]

			if modifyBaseUrl != "" {
				normalized, err := validateAndNormalizeUrl(modifyBaseUrl)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeInvalidUsage, "invalid --base-url")
				}
				selectedProfile.BaseUrl.Value = normalized
			}
			if modifyMgmntUrl != "" {
				normalized, err := validateAndNormalizeUrl(modifyMgmntUrl)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeInvalidUsage, "invalid --mgmnt-url")
				}
				selectedProfile.MgmntUrl.Value = normalized
			}
			if modifyServiceToken != "" {
				selectedProfile.ServiceToken.Value = modifyServiceToken
			}
			if workspaceSet {
				selectedProfile.Workspace.Value = modifyWorkspace
			}

			cfg.Profiles[modifyName] = selectedProfile

			err := config.SaveProfileConfig(cfg, path)
			if err != nil {
				log.WithError(err).Error("Failed to save config")
				return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
			}

			log.Infof("Profile %s modified successfully", modifyName)
		} else {
			if !utils.IsInputInteractive() {
				return clierr.New("required flags missing (--name and a field to change), cannot prompt in non-interactive mode", clierr.CodeInvalidUsage)
			}
			runModifyInteractive(cfg, path)
		}
		return nil
	},
}

func init() {
	profilesModifyCmd.Flags().StringVar(&modifyName, "name", "", "Name of the profile to modify")
	profilesModifyCmd.Flags().StringVar(&modifyBaseUrl, "base-url", "", "Base URL (default: "+config.DefaultBaseUrl+")")
	profilesModifyCmd.Flags().StringVar(&modifyMgmntUrl, "mgmnt-url", "", "Management URL (default: "+config.DefaultMgmntUrl+")")
	profilesModifyCmd.Flags().StringVar(&modifyServiceToken, "service-token", "", "Service Token")
	profilesModifyCmd.Flags().StringVar(&modifyWorkspace, "workspace", "", `Default workspace for commands run with this profile; "" removes it`)
	ProfileCmd.AddCommand(profilesModifyCmd)
}

func runModifyInteractive(cfg *config.ProfileConfig, path string) {
	var profileNames []string
	for name := range cfg.Profiles {
		profileNames = append(profileNames, name)
	}

	if len(profileNames) == 0 {
		log.Info("No profiles found. Use the command 'suprsend profile add' to add a profile.")
		return
	}

	if modifyName == "" {
		err := ask([]question{
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
		if err != nil {
			log.Error(err)
			return
		}
	}

	selectedProfile, exists := cfg.Profiles[modifyName]
	if !exists {
		log.Infof("Profile %q does not exist. Use the command 'suprsend profile list' to see all profiles.", modifyName)
		return
	}

	var questions []question

	if modifyServiceToken == "" {
		currentToken := selectedProfile.ServiceToken.Value
		maskedToken := MaskServiceToken(currentToken)
		questions = append(questions, question{
			Text: fmt.Sprintf("Service Token (current: %s, press Enter to keep): ", maskedToken),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s != "" {
					modifyServiceToken = s
				} else {
					modifyServiceToken = selectedProfile.ServiceToken.Value
				}
				return nil
			},
		})
	}

	// URL prompts. Pre-fill modifyBaseUrl / modifyMgmntUrl with the
	// current profile value (or public default if unset) BEFORE building
	// the question, so that an interrupted prompt (Ctrl+C) doesn't leave
	// the field empty and clobber the stored URL on save. The handler
	// overwrites the value when the user types a new URL.
	if modifyBaseUrl == "" {
		current := selectedProfile.BaseUrl.Value
		if current == "" {
			current = config.DefaultBaseUrl
		}
		modifyBaseUrl = current
		questions = append(questions, question{
			Text: fmt.Sprintf("Base URL [%s]: ", current),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					return nil
				}
				normalized, err := validateAndNormalizeUrl(s)
				if err != nil {
					return err
				}
				modifyBaseUrl = normalized
				return nil
			},
		})
	}
	if modifyMgmntUrl == "" {
		current := selectedProfile.MgmntUrl.Value
		if current == "" {
			current = config.DefaultMgmntUrl
		}
		modifyMgmntUrl = current
		questions = append(questions, question{
			Text: fmt.Sprintf("Management URL [%s]: ", current),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					return nil
				}
				normalized, err := validateAndNormalizeUrl(s)
				if err != nil {
					return err
				}
				modifyMgmntUrl = normalized
				return nil
			},
		})
	}

	modifyWorkspace = selectedProfile.Workspace.Value
	current := modifyWorkspace
	if current == "" {
		current = "none"
	}
	questions = append(questions, question{
		Text: fmt.Sprintf("Default workspace [%s] (Enter keeps it, \"-\" removes it): ", current),
		Handler: func(s string) error {
			s = cleanInput(s)
			switch s {
			case "":
				return nil
			case "-":
				modifyWorkspace = ""
				return nil
			}
			ws, err := normalizeProfileWorkspace(s)
			if err == nil {
				modifyWorkspace = ws
			}
			return err
		},
	})

	if err := ask(questions); err != nil {
		log.Error(err)
		return
	}

	if modifyName == "" {
		log.Error("Profile name is required")
		return
	}
	if modifyServiceToken == "" {
		log.Error("Service token is required")
		return
	}

	updatedProfile := config.Profile{
		BaseUrl:      config.ConfigString{Value: modifyBaseUrl},
		MgmntUrl:     config.ConfigString{Value: modifyMgmntUrl},
		ServiceToken: config.ConfigString{Value: modifyServiceToken},
		Workspace:    config.ConfigString{Value: modifyWorkspace},
	}

	cfg.Profiles[modifyName] = updatedProfile

	err := config.SaveProfileConfig(cfg, path)
	if err != nil {
		log.WithError(err).Error("Failed to save config")
		return
	}

	log.Infof("Profile %s modified successfully", modifyName)
}
