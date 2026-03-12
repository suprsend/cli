package profiles

import (
	"math"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
)

var listProfilesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  "List all profiles from the config. Only useful if you have a BYOC/self-hosted SuprSend instance or if you want to manage multiple accounts. Not required for moving assets between workspaces in the same account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString("config")
		if err != nil {
			log.WithError(err).Error("Couldn't find the path")
			return err
		}
		cfg, _, err := EnsureConfig(path)
		if err != nil {
			log.WithError(err).Error("Failed to load config")
			return err
		}

		var names []string
		for name := range cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)

		if len(names) == 0 {
			log.Info("No profiles found. Use 'suprsend profiles add' to add a profile")
			return nil
		}

		outputType, _ := cmd.Flags().GetString("output")
		hasBaseUrl := false
		hasMgmntUrl := false
		hasServiceToken := false

		for _, name := range names {
			profile := cfg.Profiles[name]
			if profile.BaseUrl != "" {
				hasBaseUrl = true
			}
			if profile.MgmntUrl != "" {
				hasMgmntUrl = true
			}
			if profile.ServiceToken != "" {
				hasServiceToken = true
			}
		}

		// Cleanup service token so that it is not printed fully, only the first 4 characters and the last 4 characters are printed rested are replaced with *
		for _, name := range names {
			profile := cfg.Profiles[name]
			length := len(profile.ServiceToken)
			max_cut := int(math.Min(8, float64(length)))
			profile.ServiceToken = profile.ServiceToken[:max_cut] + "*****************" + profile.ServiceToken[len(profile.ServiceToken)-4:]
			cfg.Profiles[name] = profile
		}

		if hasBaseUrl || hasMgmntUrl || hasServiceToken {
			var profileData []ProfileListItem

			for _, name := range names {
				profile := cfg.Profiles[name]
				isActive := "no"
				if name == cfg.ActiveProfile {
					isActive = "yes"
				}

				profileData = append(profileData, ProfileListItem{
					Name:         name,
					Active:       isActive,
					BaseUrl:      profile.BaseUrl,
					MgmntUrl:     profile.MgmntUrl,
					ServiceToken: profile.ServiceToken,
				})
			}

			utils.OutputData(profileData, outputType)
		} else {
			var profileData []SimpleProfileListItem

			for _, name := range names {
				isActive := "no"
				if name == cfg.ActiveProfile {
					isActive = "yes"
				}

				profileData = append(profileData, SimpleProfileListItem{
					Name:   name,
					Active: isActive,
				})
			}

			utils.OutputData(profileData, outputType)
		}
		return nil
	},
}

func init() {
	listProfilesCmd.Flags().StringP("output", "o", "pretty", "Output type: pretty, json, yaml")
	ProfileCmd.AddCommand(listProfilesCmd)
}
