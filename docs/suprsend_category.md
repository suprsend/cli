## suprsend category

Manage preference categories

### Synopsis

Manage notification preference categories. Categories organize notification preferences into a hierarchy of root categories, sections, and individual preference items.

```
suprsend category [flags]
```

### Examples

```
  suprsend category list
  suprsend category get --output json
  suprsend category pull --dir ./suprsend/categories
  suprsend category push --commit
```

### Options

```
  -h, --help                   help for category
  -s, --service-token string   Service token (default: $SUPRSEND_SERVICE_TOKEN)
  -w, --workspace string       Workspace name (e.g., staging, production) (default "staging")
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
* [suprsend category commit](suprsend_category_commit.md)	 - Commit categories
* [suprsend category get](suprsend_category_get.md)	 - Get categories and translations
* [suprsend category list](suprsend_category_list.md)	 - List categories
* [suprsend category pull](suprsend_category_pull.md)	 - Pull categories from a workspace
* [suprsend category push](suprsend_category_push.md)	 - Push categories to a workspace
* [suprsend category translation](suprsend_category_translation.md)	 - Manage preference category translations

