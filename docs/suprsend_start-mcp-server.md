## suprsend start-mcp-server

Start SuprSend MCP server

### Synopsis

Start an MCP (Model Context Protocol) server that exposes SuprSend tools for AI assistants.

Built-in tool categories: users (get, upsert, preferences, subscriptions), objects (get, upsert, preferences, subscriptions), tenants (get, upsert, preferences), workflows (list), and documentation (search, fetch). Use --tools to select categories (e.g., --tools=users.*,tenants.*) or specific tools (e.g., --tools=users.get,tenants.get_all).

Use --events and --workflows to dynamically register tools that trigger specific events or workflows by slug. Both default to none — pass 'all' to register tools for every event/workflow in the workspace, or a comma-separated list of slugs to register specific ones.

Transports: stdio (default, for CLI/IDE integrations), sse (listens on :8080/sse), http (listens on :8080/).

```
suprsend start-mcp-server [flags]
```

### Options

```
  -e, --events string      Event tools to register: all, none, or comma-separated event names (tag: prefix reserved for future use) (default "none")
  -h, --help               help for start-mcp-server
  -T, --tools string       Tools to expose: all, none, or comma-separated tool names (default "all")
  -t, --transport string   Server transport: stdio, sse, or http (default "stdio")
  -W, --workflows string   Workflow tools to register: all, none, comma-separated slugs, or tag:<tag> entries (e.g. tag:onboarding,tag:transactional) (default "none")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
      --no-color           Disable color output (default: $NO_COLOR)
  -o, --output string      Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet              Suppress info/warn output (errors are still shown)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure
* [suprsend start-mcp-server list-tools](suprsend_start-mcp-server_list-tools.md)	 - List all the tools supported by the server

