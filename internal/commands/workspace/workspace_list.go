package workspace

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
)

type WorkspaceTableRow struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Mode        string `json:"mode"`
	Description string `json:"description"`
	IsExpired   bool   `json:"is_expired"`
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	Long:  "List all SuprSend workspaces accessible with the current service token. Returns workspace name, slug, mode, and description.",
	Example: `  # List workspaces
  suprsend workspace list

  # Paginate results
  suprsend workspace list --limit 5 --offset 0

  # JSON output
  suprsend workspace list --output json`,
	Annotations: map[string]string{
		"skills:tip:output": "Use `-o json` for machine-readable JSON output, `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := utils.NewSpinner("Loading...")
		mgmntClient := utils.GetSuprSendMgmntClient()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")

		workspaces, err := mgmntClient.ListWorkspaces(cmd.Context(), limit, offset)
		if err != nil {
			log.WithError(err).Error("Couldn't fetch workspaces")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}

		spinner.Stop(fmt.Sprintf("Listed %d workspaces", len(workspaces.Results)))
		outputType, _ := cmd.Flags().GetString("output")

		if len(workspaces.Results) == 0 && utils.IsOutputPiped() {
			utils.OutputData([]any{}, outputType)
			return nil
		}

		var rows []WorkspaceTableRow
		for _, ws := range workspaces.Results {
			rows = append(rows, WorkspaceTableRow{
				Name:        ws.Name,
				Slug:        ws.Slug,
				Mode:        ws.Mode,
				Description: ws.Description,
				IsExpired:   ws.IsExpired,
			})
		}

		utils.OutputData(rows, outputType)
		return nil
	},
}

func init() {
	workspaceListCmd.Flags().IntP("limit", "l", 20, "Maximum number of workspaces to return")
	workspaceListCmd.Flags().Int("offset", 0, "Number of workspaces to skip for pagination")
}
