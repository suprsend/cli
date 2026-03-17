# suprsend start-mcp-server

Start SuprSend MCP server

Start an MCP (Model Context Protocol) server that exposes SuprSend tools for AI assistants.

Built-in tool categories: users (get, upsert, preferences, subscriptions), objects (get, upsert, preferences, subscriptions), tenants (get, upsert, preferences), workflows (list), and documentation (search, fetch). Use --tools to select categories (e.g., --tools=users.*,tenants.*) or specific tools (e.g., --tools=users.get,tenants.get_all).

Use --events and --workflows to dynamically register tools that trigger specific events or workflows by slug. Both default to none — pass 'all' to register tools for every event/workflow in the workspace, or a comma-separated list of slugs to register specific ones.

Transports: stdio (default, for CLI/IDE integrations), sse (listens on :8080/sse), http (listens on :8080/).

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

