/*
Copyright © 2025 SuprSend
*/
package commands

import (
	"strings"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/suprsend/cli/internal/commands/profiles"
	"github.com/suprsend/cli/internal/config"
	toolset "github.com/suprsend/cli/internal/tools"
	"github.com/suprsend/cli/internal/utils"
	"go.szostok.io/version"
)

var (
	transport string
	tools     string
	events    string
	workflows string
)

func getSelectedTools(toolsFlag string) ([]*toolset.Tool, error) {
	if toolsFlag == "none" {
		return []*toolset.Tool{}, nil
	}
	// selected tools
	selected := []*toolset.Tool{}
	supportedTools := toolset.GetAllTools()
	// if toolsFlag is "all", return all the tools
	if toolsFlag == "all" {
		return supportedTools, nil
	}
	// get the tools mentioned in toolsFlag
	tools := strings.Split(toolsFlag, ",")

	// if tool name is `type`.* include all the tools that have same type
	for _, tool := range tools {
		if strings.Contains(tool, ".*") {
			toolType := strings.Split(tool, ".*")[0]
			for _, t := range supportedTools {
				if t.Type == toolType {
					selected = append(selected, t)
				}
			}
		} else {
			for _, t := range supportedTools {
				if t.Name == tool {
					selected = append(selected, t)
				}
			}
		}
	}
	return selected, nil
}

// startMcpServerCmd represents the startMcpServer command
var startMcpServerCmd = &cobra.Command{
	Use:   "start-mcp-server",
	Short: "Start SuprSend MCP server",
	Long: `Start SuprSend MCP server.
This server will handle all the requests from user about SuprSend capabilities and data.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		conf := config.Cfg
		workspace := conf.Workspace
		serviceToken := getServiceTokenWithPriority()
		conf.ServiceToken = serviceToken
		utils.InitSDKWithUrls(
			conf.ServiceToken,
			profiles.GetResolvedBaseUrl(),
			profiles.GetResolvedMgmntUrl(),
			viper.GetBool("debug"),
		)
		if err := toolset.RegisterDynamicEventsTools(workspace, events); err != nil {
			log.Warnf("Failed to register event tools in mcp: %v", err)
		}
		if err := toolset.RegisterDynamicWorkflowTools(workspace, workflows); err != nil {
			log.Warnf("Failed to register workflow tools in mcp: %v", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		selectedTools, err := getSelectedTools(tools)
		if err != nil {
			log.Fatalf("%v", err)
		}
		selectedEvents := toolset.GetAllEvents()
		selectedWorkflows := toolset.GetAllWorkflows()

		selectedTools = append(selectedTools, selectedEvents...)
		selectedTools = append(selectedTools, selectedWorkflows...)

		// Print a readable string representation of selectedTools
		var toolStrs []string
		for _, t := range selectedTools {
			toolStrs = append(toolStrs, t.Type+":"+t.Name)
		}
		log.Infof("Selected tools: [%s]", strings.Join(toolStrs, ", "))
		info := version.Get()

		mcpServer := server.NewMCPServer(
			"SuprSend",
			info.Version,
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
			server.WithLogging(),
			server.WithRecovery(),
		)

		for _, t := range selectedTools {
			mcpServer.AddTool(t.MCPTool, t.Handler)
		}

		switch transport {
		case "stdio":
			if err := server.ServeStdio(mcpServer); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		case "sse":
			utils.Banner(info.Version)
			sseServer := server.NewSSEServer(mcpServer)
			log.Printf("SSE server listening on :8080/sse")
			if err := sseServer.Start(":8080"); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		case "http":
			utils.Banner(info.Version)
			httpServer := server.NewStreamableHTTPServer(mcpServer, server.WithEndpointPath("/"))
			log.Printf("HTTP server listening on :8080/")
			if err := httpServer.Start(":8080"); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		default:
			log.Fatalf("Invalid transport: %s. Valid transports are stdio/sse/http", transport)
		}
	},
}

var listToolsCmd = &cobra.Command{
	Use:   "list-tools",
	Short: "List all the tools supported by the server",
	Run: func(cmd *cobra.Command, args []string) {
		type toolListResponse struct {
			Tool_Type        string `json:"tool_type"`
			Tool_Name        string `json:"tool_name"`
			Tool_Description string `json:"tool_description"`
		}
		var resp []toolListResponse
		for _, t := range toolset.GetAllTools() {
			resp = append(resp, toolListResponse{Tool_Type: t.Type, Tool_Name: t.Name, Tool_Description: t.Description})
		}
		for _, t := range toolset.GetAllEvents() {
			resp = append(resp, toolListResponse{Tool_Type: t.Type, Tool_Name: t.Name, Tool_Description: t.Description})
		}
		for _, t := range toolset.GetAllWorkflows() {
			resp = append(resp, toolListResponse{Tool_Type: t.Type, Tool_Name: t.Name, Tool_Description: t.Description})
		}
		outputType, _ := cmd.Flags().GetString("output")
		utils.OutputData(resp, outputType)
	},
}

func init() {
	startMcpServerCmd.AddCommand(listToolsCmd)
	rootCmd.AddCommand(startMcpServerCmd)

	startMcpServerCmd.PersistentFlags().StringVarP(&transport, "transport", "t", "stdio", "The transport to use for the MCP server. Can be stdio/sse/http.")
	startMcpServerCmd.PersistentFlags().StringVarP(&tools, "tools", "T", "all", "The types of tools to use. Can be either 'all'/'none' or comma separated list of tool names.")
	startMcpServerCmd.PersistentFlags().StringVarP(&events, "events", "e", "none", "The types of events to use. Can be either 'all'/'none' or comma separated list of event slugs.")
	startMcpServerCmd.PersistentFlags().StringVarP(&workflows, "workflows", "W", "none", "The types of workflows to use. Can be either 'all'/'none' or comma separated list of workflow slugs.")
	startMcpServerCmd.PersistentFlags().StringP("service-token", "s", "", "Service token (default: $SUPRSEND_SERVICE_TOKEN)")
	viper.BindPFlag("service_token", startMcpServerCmd.PersistentFlags().Lookup("service-token"))
}
