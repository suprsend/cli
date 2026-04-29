# suprsend completion bash

Generate the autocompletion script for bash

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(suprsend completion bash)

To load completions for every new session, execute once:

#### Linux:

	suprsend completion bash > /etc/bash_completion.d/suprsend

#### macOS:

	suprsend completion bash > $(brew --prefix)/etc/bash_completion.d/suprsend

You will need to start a new shell for this setup to take effect.


```
suprsend completion bash
```

### Options

```
  -h, --help              help for bash
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

