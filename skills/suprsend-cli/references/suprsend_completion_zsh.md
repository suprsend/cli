# suprsend completion zsh

Generate the autocompletion script for zsh

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(suprsend completion zsh)

To load completions for every new session, execute once:

#### Linux:

	suprsend completion zsh > "${fpath[1]}/_suprsend"

#### macOS:

	suprsend completion zsh > $(brew --prefix)/share/zsh/site-functions/_suprsend

You will need to start a new shell for this setup to take effect.


```
suprsend completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

