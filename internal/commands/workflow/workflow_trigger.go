package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/suprsend-go"
	"github.com/yarlson/pin"
)

var workflowTrigger = &cobra.Command{
	Use:   "trigger [<slug>]",
	Short: "Trigger a specific workflow",
	Long:  "Trigger a specific workflow by passing a slug as a positional argument or via --slug.",
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
			return err
		}
		var p *pin.Pin
		if !utils.IsOutputPiped() {
			p = pin.New("Triggering workflow...",
				pin.WithSpinnerColor(pin.ColorCyan),
				pin.WithTextColor(pin.ColorYellow),
			)
			cancel := p.Start(context.Background())
			defer cancel()
		}
		wfRequestBody, err := os.ReadFile(path)
		if err != nil {
			log.WithError(err).Error("Error reading workflow file")
			return err
		}
		wfRequestBodyMap := make(map[string]any)
		err = json.Unmarshal(wfRequestBody, &wfRequestBodyMap)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling workflow file")
			return err
		}

		wf := &suprsend.WorkflowTriggerRequest{
			Body:     wfRequestBodyMap,
			TenantId: tenantId,
		}
		_, err = wsClient.Workflows.Trigger(wf)
		if err != nil {
			log.WithError(err).Error("Error triggering workflow")
			return err
		}
		if p != nil {
			p.Stop(fmt.Sprintf("Successfully triggered workflow '%s'", slug))
		} else {
			fmt.Fprintf(os.Stdout, "Successfully triggered workflow '%s'", slug)
		}
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
