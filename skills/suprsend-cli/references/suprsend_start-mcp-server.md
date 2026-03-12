# suprsend start-mcp-server

Start SuprSend MCP server

Start SuprSend MCP server.
This server will handle all the requests from user about SuprSend capabilities and data.

```
suprsend start-mcp-server [flags]
```

### Options

```
  -e, --events string          The types of events to use. Can be either 'all'/'none' or comma separated list of event slugs. (default "none")
  -h, --help                   help for start-mcp-server
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -T, --tools string           The types of tools to use. Can be either 'all'/'none' or comma separated list of tool names. (default "all")
  -t, --transport string       The transport to use for the MCP server. Can be stdio/sse/http. (default "stdio")
  -W, --workflows string       The types of workflows to use. Can be either 'all'/'none' or comma separated list of workflow slugs. (default "none")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

