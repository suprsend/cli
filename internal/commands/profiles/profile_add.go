package profiles

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sabouaram/cobra_ui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	Long:  "Add a new profile to the configs. Pass --base-url / --mgmnt-url to point the profile at non-default URLs (BYOC, staging, pre-prod, dev, ...); omit them to use the public SuprSend Cloud defaults.",
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("config")

		cfg, path, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load or create config")
			return
		}

		if addName != "" && addServiceToken != "" {
			// Scripted path: no prompts. Validate URLs if provided, fall
			// back to public defaults if not.
			if addBaseUrl != "" {
				normalized, err := validateAndNormalizeUrl(addBaseUrl)
				if err != nil {
					log.WithError(err).Error("Invalid --base-url")
					return
				}
				addBaseUrl = normalized
			} else {
				addBaseUrl = DefaultBaseUrl
			}
			if addMgmntUrl != "" {
				normalized, err := validateAndNormalizeUrl(addMgmntUrl)
				if err != nil {
					log.WithError(err).Error("Invalid --mgmnt-url")
					return
				}
				addMgmntUrl = normalized
			} else {
				addMgmntUrl = DefaultMgmntUrl
			}

			cfg.Profiles[addName] = Profile{
				BaseUrl:      addBaseUrl,
				MgmntUrl:     addMgmntUrl,
				ServiceToken: addServiceToken,
			}

			err := SaveConfig(cfg, path)
			if err != nil {
				log.WithError(err).Error("Failed to save config")
				return
			}

			log.Infof("Profile %s added successfully", addName)
		} else {
			runAddInteractive(cfg, path)
		}
	},
}

func init() {
	profilesAddCmd.Flags().StringVar(&addName, "name", "", "Name of the profile (required)")
	profilesAddCmd.Flags().StringVar(&addBaseUrl, "base-url", "", "Base URL (default: "+DefaultBaseUrl+")")
	profilesAddCmd.Flags().StringVar(&addMgmntUrl, "mgmnt-url", "", "Management URL (default: "+DefaultMgmntUrl+")")
	profilesAddCmd.Flags().StringVar(&addServiceToken, "service-token", "", "Service token (required)")
	ProfileCmd.AddCommand(profilesAddCmd)
}

func runAddInteractive(cfg *Config, path string) {
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
			Text: fmt.Sprintf("Base URL [%s]: ", DefaultBaseUrl),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					addBaseUrl = DefaultBaseUrl
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
			Text: fmt.Sprintf("Management URL [%s]: ", DefaultMgmntUrl),
			Handler: func(s string) error {
				s = cleanInput(s)
				if s == "" {
					addMgmntUrl = DefaultMgmntUrl
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

	cfg.Profiles[addName] = Profile{
		BaseUrl:      addBaseUrl,
		MgmntUrl:     addMgmntUrl,
		ServiceToken: addServiceToken,
	}

	err := SaveConfig(cfg, path)
	if err != nil {
		log.WithError(err).Error("Failed to save config")
		return
	}

	log.Infof("Profile %s added successfully!\n", addName)
}
