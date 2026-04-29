# suprsend completion fish

Generate the autocompletion script for fish

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	suprsend completion fish | source

To load completions for every new session, execute once:

	suprsend completion fish > ~/.config/fish/completions/suprsend.fish

You will need to start a new shell for this setup to take effect.


```
suprsend completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string          config file (default: $HOME/.suprsend.yaml)
      --no-color               Disable color output (default: $NO_COLOR)
  -o, --output string          Output format: pretty, json, or yaml (default "pretty")
  -q, --quiet                  Suppress info/warn output (errors are still shown)
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -v, --verbosity string       Log level (debug, info, warn, error, fatal, panic) (default "info")
```

