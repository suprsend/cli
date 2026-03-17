## suprsend translation

Manage Translations

### Synopsis

Manage template translations. Subcommands let you list, pull, push, and commit translations for notification templates.

```
suprsend translation [flags]
```

### Options

```
  -h, --help                   help for translation
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
```

### Options inherited from parent commands

```
      --config string      config file (default: $HOME/.suprsend.yaml)
  -n, --no-color           Disable color output (default: $NO_COLOR)
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic) (default "info")
```

### SEE ALSO

* [suprsend](suprsend.md)	 - CLI to interact with SuprSend, a Notification Infrastructure
* [suprsend translation commit](suprsend_translation_commit.md)	 - Commit translation
* [suprsend translation get](suprsend_translation_get.md)	 - Get translations
* [suprsend translation list](suprsend_translation_list.md)	 - List Translations
* [suprsend translation pull](suprsend_translation_pull.md)	 - Pull Translation files
* [suprsend translation push](suprsend_translation_push.md)	 - Push translation files to a workspace

