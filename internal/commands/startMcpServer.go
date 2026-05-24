/*
Copyright © 2025 SuprSend
*/
package commands

import (
	"strings"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/config"
	legacysdk "github.com/suprsend/cli/internal/mcpsdk/legacy"
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
	Long: `Start an MCP (Model Context Protocol) server that exposes SuprSend tools for AI assistants.

Built-in tool categories: users (get, upsert, preferences, subscriptions), objects (get, upsert, preferences, subscriptions), tenants (get, upsert, preferences), workflows (list), and documentation (search, fetch). Use --tools to select categories (e.g., --tools=users.*,tenants.*) or specific tools (e.g., --tools=users.get,tenants.get_all).

Use --events and --workflows to dynamically register tools that trigger specific events or workflows by slug. Both default to none — pass 'all' to register tools for every event/workflow in the workspace, or a comma-separated list of slugs to register specific ones.

Transports: stdio (default, for CLI/IDE integrations), sse (listens on :8080/sse), http (listens on :8080/).`,
	Annotations: map[string]string{
		"skills:tip.a-auth":          "Requires `SUPRSEND_SERVICE_TOKEN` env var or an active profile (`suprsend profile use <name>`).",
		"skills:tip.b-inspect-tools": "Run `suprsend start-mcp-server list-tools` to inspect the schema (tool names + descriptions) before wiring up an MCP client.",
		"skills:tip.c-transport":     "stdio is right for IDE/CLI integrations; switch to `--transport sse` or `--transport http` for network-accessible deployments (both listen on :8080).",
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
			return err
		}
		conf := config.Cfg
		// Dynamic registration runs for both `start-mcp-server` and
		// `start-mcp-server list-tools` so the listing reflects what the
		// real server would expose. Selectors default to "none", so users
		// who don't pass --workflows / --events pay no API cost.
		if err := toolset.RegisterDynamicEventsTools(conf.Workspace.Value, events); err != nil {
			log.Warnf("Failed to register event tools in mcp: %v", err)
		}
		if err := toolset.RegisterDynamicWorkflowTools(conf.Workspace.Value, workflows); err != nil {
			log.Warnf("Failed to register workflow tools in mcp: %v", err)
		}
		return nil
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
			if t.Tool != nil {
				// Ported: registered via mcpsdk abstraction → legacy adapter.
				legacysdk.Register(mcpServer, t.Tool)
			} else {
				// Un-ported: still on raw mark3labs types. Keeps the CLI's tool
				// surface complete during the Phase 5 incremental ports.
				mcpServer.AddTool(t.MCPTool, t.Handler)
			}
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
	Long:  "List all available MCP tools with their type, name, and description. Includes built-in tools and any dynamically registered event/workflow trigger tools. Use this to discover tool names for the --tools flag.",
	Run: func(cmd *cobra.Command, args []string) {
		type toolListResponse struct {
			Tool_Type        string `json:"tool_type"`
			Tool_Name        string `json:"tool_name"`
			Tool_Description string `json:"tool_description"`
		}
		// Dual-field envelope (Phase 4–5 transition): ported tool files
		// populate t.Tool; un-ported files still populate t.MCPTool + t.Name.
		// Read from whichever is set so the listing stays accurate while the
		// per-file ports land. Removed in Task 5.x once the envelope collapses
		// to a single *mcpsdk.Tool.
		describe := func(t *toolset.Tool) toolListResponse {
			if t.Tool != nil {
				return toolListResponse{Tool_Type: t.Type, Tool_Name: t.Tool.Name, Tool_Description: t.Tool.Description}
			}
			return toolListResponse{Tool_Type: t.Type, Tool_Name: t.Name, Tool_Description: t.MCPTool.Description}
		}
		var resp []toolListResponse
		for _, t := range toolset.GetAllTools() {
			resp = append(resp, describe(t))
		}
		for _, t := range toolset.GetAllEvents() {
			resp = append(resp, describe(t))
		}
		for _, t := range toolset.GetAllWorkflows() {
			resp = append(resp, describe(t))
		}
		outputType, _ := cmd.Flags().GetString("output")
		utils.OutputData(resp, outputType)
	},
}

func init() {
	startMcpServerCmd.AddCommand(listToolsCmd)
	rootCmd.AddCommand(startMcpServerCmd)

	startMcpServerCmd.PersistentFlags().StringVarP(&transport, "transport", "t", "stdio", "Server transport: stdio, sse, or http")
	startMcpServerCmd.PersistentFlags().StringVarP(&tools, "tools", "T", "all", "Tools to expose: all, none, or comma-separated tool names")
	startMcpServerCmd.PersistentFlags().StringVarP(&events, "events", "e", "none", "Event tools to register: all, none, or comma-separated event names (tag: prefix reserved for future use)")
	startMcpServerCmd.PersistentFlags().StringVarP(&workflows, "workflows", "W", "none", "Workflow tools to register: all, none, comma-separated slugs, or tag:<tag> entries (e.g. tag:onboarding,tag:transactional)")

	listToolsCmd.Flags().StringP("output", "o", "pretty", "Output format: pretty, json, or yaml")
}
