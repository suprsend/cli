# suprsend start-mcp-server list-tools

List all the tools supported by the server

List all available MCP tools with their type, name, and description. Includes built-in tools and any dynamically registered event/workflow trigger tools. Use this to discover tool names for the --tools flag.

```
suprsend start-mcp-server list-tools [flags]
```

### Options

```
  -h, --help            help for list-tools
  -o, --output string   Output format: pretty, json, or yaml (default "pretty")
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -e, --events string          Event tools to register: all, none, or comma-separated event names (tag: prefix reserved for future use) (default "none")
      --no-color               Disable color output (default: $NO_COLOR)
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -T, --tools string           Tools to expose: all, none, or comma-separated tool names (default "all")
  -t, --transport string       Server transport: stdio, sse, or http (default "stdio")
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -W, --workflows string       Workflow tools to register: all, none, comma-separated slugs, or tag:<tag> entries (e.g. tag:onboarding,tag:transactional) (default "none")
```

