package profiles

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sabouaram/cobra_ui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/utils"
)

var (
	addName         string
	addBaseUrl      string
	addMgmntUrl     string
	addServiceToken string
)

var profilesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	Long:  "Add a new profile to the configs. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load or create config")
			return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
		}

		if addName != "" && addServiceToken != "" {
			// Scripted path: no prompts. Validate URLs if provided, fall
			// back to public defaults if not.
			if addBaseUrl != "" {
				normalized, err := validateAndNormalizeUrl(addBaseUrl)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeInvalidUsage, "invalid --base-url")
				}
				addBaseUrl = normalized
			} else {
				addBaseUrl = config.DefaultBaseUrl
			}
			if addMgmntUrl != "" {
				normalized, err := validateAndNormalizeUrl(addMgmntUrl)
				if err != nil {
					return clierr.Wrap(err, clierr.CodeInvalidUsage, "invalid --mgmnt-url")
				}
				addMgmntUrl = normalized
			} else {
				addMgmntUrl = config.DefaultMgmntUrl
			}

			cfg.Profiles[addName] = config.Profile{
				BaseUrl:      config.ConfigString{Value: addBaseUrl},
				MgmntUrl:     config.ConfigString{Value: addMgmntUrl},
				ServiceToken: config.ConfigString{Value: addServiceToken},
			}

			err := config.SaveProfileConfig(cfg, path)
			if err != nil {
				log.WithError(err).Error("Failed to save config")
				return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
			}

			log.Infof("Profile %s added successfully", addName)
		} else {
			if !utils.IsInputInteractive() {
				return clierr.New("required flags missing (--name, --service-token), cannot prompt in non-interactive mode", clierr.CodeInvalidUsage)
			}
			runAddInteractive(cfg, path)
		}
		return nil
	},
}

func init() {
	profilesAddCmd.Flags().StringVar(&addName, "name", "", "Name of the profile (required)")
	profilesAddCmd.Flags().StringVar(&addBaseUrl, "base-url", "", "Base URL (default: "+config.DefaultBaseUrl+")")
	profilesAddCmd.Flags().StringVar(&addMgmntUrl, "mgmnt-url", "", "Management URL (default: "+config.DefaultMgmntUrl+")")
	profilesAddCmd.Flags().StringVar(&addServiceToken, "service-token", "", "Service token (required)")
	ProfileCmd.AddCommand(profilesAddCmd)
}

func runAddInteractive(cfg *config.ProfileConfig, path string) {
	ui := cobra_ui.New()
	var questions []cobra_ui.Question

	if addName == "" {
		questions = append(questions, cobra_ui.Question{
			Text: "Name: ",
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					return fmt.Errorf("profile name cannot be empty")
				}
				if _, exists := cfg.Profiles[s]; exists {
					return fmt.Errorf("profile '%s' already exists", s)
				}
				addName = s
				return nil
			},
		})
	}

	if addServiceToken == "" {
		questions = append(questions, cobra_ui.Question{
			Text: "Service Token: ",
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					return fmt.Errorf("service token cannot be empty")
				}
				addServiceToken = s
				return nil
			},
		})
	}

	// URL prompts. cobra_ui has no native "default value" support, so we
	// embed the default in the prompt text and let the handler treat
	// empty input as "accept the default". Typed URL is validated and
	// stored. The same flow covers BYOC, staging / pre-prod / dev — any
	// non-default endpoint works the same way.
	if addBaseUrl == "" {
		questions = append(questions, cobra_ui.Question{
			Text: fmt.Sprintf("Base URL [%s]: ", config.DefaultBaseUrl),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					addBaseUrl = config.DefaultBaseUrl
					return nil
				}
				normalized, err := validateAndNormalizeUrl(s)
				if err != nil {
					return err
				}
				addBaseUrl = normalized
				return nil
			},
		})
	}
	if addMgmntUrl == "" {
		questions = append(questions, cobra_ui.Question{
			Text: fmt.Sprintf("Management URL [%s]: ", config.DefaultMgmntUrl),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					addMgmntUrl = config.DefaultMgmntUrl
					return nil
				}
				normalized, err := validateAndNormalizeUrl(s)
				if err != nil {
					return err
				}
				addMgmntUrl = normalized
				return nil
			},
		})
	}

	if len(questions) > 0 {
		ui.SetQuestions(questions)
		ui.RunInteractiveUI()
	}

	if addName == "" {
		log.Error("Profile name is required")
		return
	}
	if addServiceToken == "" {
		log.Error("Service token is required")
		return
	}

	fmt.Println("\n Profile Summary:")
	fmt.Printf("   Name: %s\n", addName)
	fmt.Printf("   Service Token: [HIDDEN]\n")
	fmt.Printf("   Base URL: %s\n", addBaseUrl)
	fmt.Printf("   Management URL: %s\n", addMgmntUrl)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n Add this profile? (Y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			log.WithError(err).Error("Failed to read input")
			return
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			break
		}
		if response == "n" || response == "no" || response == "" {
			log.Infof("Profile creation cancelled")
			return
		}

		log.Infof(" Please enter 'y' for yes or 'n' for no")
	}

	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = addName
		log.Infof("Set '%s' as the active profile", addName)
	}

	cfg.Profiles[addName] = config.Profile{
		BaseUrl:      config.ConfigString{Value: addBaseUrl},
		MgmntUrl:     config.ConfigString{Value: addMgmntUrl},
		ServiceToken: config.ConfigString{Value: addServiceToken},
	}

	err := config.SaveProfileConfig(cfg, path)
	if err != nil {
		log.WithError(err).Error("Failed to save config")
		return
	}

	log.Infof("Profile %s added successfully!\n", addName)
}
