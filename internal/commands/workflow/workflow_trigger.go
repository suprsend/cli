package workflow

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/suprsend-go"
)

var workflowTrigger = &cobra.Command{
	Use:   "trigger [<slug>]",
	Short: "Trigger a specific workflow",
	Long:  "Trigger a specific workflow by passing a slug as a positional argument or via --slug.",
	Example: `  # Trigger a workflow by slug (positional)
  suprsend workflow trigger welcome

  # Trigger using the flag form
  suprsend workflow trigger --slug welcome

  # Trigger with a custom payload from a file
  suprsend workflow trigger welcome --path ./payload.json`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := utils.ResolveSlug(cmd, args)
		if slug == "" {
			log.Error("workflow slug is required: provide it as a positional argument or via --slug")
			return fmt.Errorf("workflow slug is required: provide it as a positional argument or via --slug")
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		tenantId, _ := cmd.Flags().GetString("tenant")
		wsClient, err := utils.GetSuprSendWorkspaceClient(workspace)
		path, _ := cmd.Flags().GetString("path")
		if err != nil {
			log.WithError(err).Error("Error getting workspace client")
			return clierr.Wrap(err, clierr.CodeConfigInvalid, "")
		}
		spinner := utils.NewSpinner("Triggering workflow...")
		wfRequestBody, err := os.ReadFile(path)
		if err != nil {
			log.WithError(err).Error("Error reading workflow file")
			return clierr.Wrap(err, clierr.CodeFileNotFound, "")
		}
		wfRequestBodyMap := make(map[string]any)
		err = json.Unmarshal(wfRequestBody, &wfRequestBodyMap)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling workflow file")
			return clierr.Wrap(err, clierr.CodeFileParseFailed, "")
		}

		wf := &suprsend.WorkflowTriggerRequest{
			Body:     wfRequestBodyMap,
			TenantId: tenantId,
		}
		_, err = wsClient.Workflows.Trigger(wf)
		if err != nil {
			log.WithError(err).Error("Error triggering workflow")
			return clierr.Wrap(err, clierr.CodeAPIInternal, "")
		}
		spinner.Stop(fmt.Sprintf("Successfully triggered workflow '%s'", slug))
		return nil
	},
}

func init() {
	workflowTrigger.Flags().StringP("slug", "g", "", "Workflow slug")
	workflowTrigger.PersistentFlags().String("path", "", "json body to trigger the wf")
	workflowTrigger.MarkFlagRequired("path")
	workflowTrigger.PersistentFlags().String("tenant", "", "tenant id to pass in body")
	// WorkflowCmd.AddCommand(workflowTrigger)
}
