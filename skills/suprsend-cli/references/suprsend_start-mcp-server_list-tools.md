# suprsend start-mcp-server list-tools

List all the tools supported by the server

```
suprsend start-mcp-server list-tools [flags]
```

### Options

```
  -h, --help   help for list-tools
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
  -e, --events string          The types of events to use. Can be either 'all'/'none' or comma separated list of event slugs. (default "none")
  -n, --no-color               Disable color output (default: $NO_COLOR)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -T, --tools string           The types of tools to use. Can be either 'all'/'none' or comma separated list of tool names. (default "all")
  -t, --transport string       The transport to use for the MCP server. Can be stdio/sse/http. (default "stdio")
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
  -W, --workflows string       The types of workflows to use. Can be either 'all'/'none' or comma separated list of workflow slugs. (default "none")
```

