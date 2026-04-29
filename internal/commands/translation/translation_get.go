package translation

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

var translationGetCmd = &cobra.Command{
	Use:   "get [<filename>]",
	Short: "Get a single translation file",
	Long:  "Retrieve a single template translation from a workspace by filename. Pass the filename as a positional argument or via --slug. Returns the translation's content (the localized string map). Use --mode to switch between draft and live versions.",
	Example: `  # Get a translation by filename (positional)
  suprsend translation get candidate.es.json

  # Get using the flag form
  suprsend translation get --slug candidate.es.json

  # Get from draft mode
  suprsend translation get candidate.es.json --mode draft

  # Get with YAML output
  suprsend translation get candidate.es.json --output yaml`,
	Args: cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := utils.ResolveSlug(cmd, args)
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		outputType, _ := cmd.Flags().GetString("output")
		if err := utils.ValidateOutputType(outputType, "json", "yaml"); err != nil {
			return err
		}
		if filename == "" {
			return clierr.New("filename is required: provide it as a positional argument or via --slug (e.g. translation get candidate.es.json)", clierr.CodeInvalidUsage)
		}

		mgmntClient := utils.GetSuprSendMgmntClient()
		spinner := utils.NewSpinner("Getting translation...")

		// The management API doesn't expose a single-translation read
		// endpoint, so we list with include_content=true and filter
		// client-side. Pull a wide page so any plausible workspace fits;
		// the response is bounded by the workspace's translation count.
		translations, err := mgmntClient.ListTranslations(workspace, mode, "true", 1000, 0)
		if err != nil {
			spinner.Stop("")
			log.WithError(err).Errorf("Error getting translations")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		// Match against either the exact filename or the stem (locale) so
		// callers can pass "candidate.es.json" or "candidate.es" — same
		// looseness the pull side uses when writing files to disk.
		stem := strings.TrimSuffix(filename, ".json")
		for _, t := range translations.Results {
			if t.FileName == filename || strings.TrimSuffix(t.FileName, ".json") == stem {
				spinner.Stop(fmt.Sprintf("Got translation %s", t.FileName))
				utils.OutputData(t, outputType)
				return nil
			}
		}

		spinner.Stop("")
		return clierr.New(fmt.Sprintf("Translation '%s' not found in workspace '%s' (mode=%s)", filename, workspace, mode), clierr.CodeAPINotFound)
	},
}

func init() {
	translationGetCmd.PersistentFlags().StringP("slug", "g", "", "Translation filename, e.g. candidate.es.json")
	translationGetCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	translationGetCmd.PersistentFlags().StringP("output", "o", "json", "Output format: json or yaml")
	TranslationCmd.AddCommand(translationGetCmd)
}
