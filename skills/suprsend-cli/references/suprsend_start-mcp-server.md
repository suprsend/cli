# suprsend start-mcp-server

Start SuprSend MCP server

Start SuprSend MCP server that exposes tools for AI assistants to interact with your SuprSend workspace.
Supports stdio, SSE, and HTTP transports. Use --tools to select which tools to expose, and --events/--workflows to dynamically register event/workflow-specific tools.

```
suprsend start-mcp-server [flags]
```

### Options

```
  -e, --events string      Event tools to register: all, none, or comma-separated event slugs (default "none")
  -h, --help               help for start-mcp-server
  -T, --tools string       Tools to expose: all, none, or comma-separated tool names (default "all")
  -t, --transport string   Server transport: stdio, sse, or http (default "stdio")
  -W, --workflows string   Workflow tools to register: all, none, or comma-separated workflow slugs (default "none")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

