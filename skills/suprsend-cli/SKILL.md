---
name: suprsend-cli
description: "SuprSend CLI is a command-line interface tool for managing your SuprSend account and resources. It provides a convenient way to interact with the SuprSend API, allowing you to perform various operations such as managing workspaces, users, workflow, templates and more."
---

# suprsend

SuprSend is a robust notification infrastructure that helps you deploy multi-channel product notifications effortlessly and take care of user experience.

This CLI lets you interact with your SuprSend workspace and do actions like fetching/modifying template, workflows etc.

## Notes

- Commands that return data (list, get) support `-o json` for machine-readable JSON output and `-o yaml` for YAML. Default `-o pretty` outputs a human-friendly table.
- The `profile` command and its subcommands (add, list, modify, remove, use) are only needed for self-hosted/BYOC SuprSend instances or managing multiple accounts. SaaS users do not need them. Profiles are not used for switching between workspaces within the same account; use the `--workspace` flag for that.

## Available Commands

- [`suprsend category`](references/suprsend_category.md) - Manage preference categories
- [`suprsend category commit`](references/suprsend_category_commit.md) - Commit categories
- [`suprsend category get`](references/suprsend_category_get.md) - Get categories and translations
- [`suprsend category list`](references/suprsend_category_list.md) - List categories
- [`suprsend category pull`](references/suprsend_category_pull.md) - Pull categories from a workspace
- [`suprsend category push`](references/suprsend_category_push.md) - Push categories to a workspace
- [`suprsend category translation`](references/suprsend_category_translation.md) - Manage preference category translations
- [`suprsend category translation list`](references/suprsend_category_translation_list.md) - List preference translations
- [`suprsend category translation pull`](references/suprsend_category_translation_pull.md) - Pull preference translations
- [`suprsend category translation push`](references/suprsend_category_translation_push.md) - Push preference translations
- [`suprsend completion`](references/suprsend_completion.md) - Generate the autocompletion script for the specified shell
- [`suprsend completion bash`](references/suprsend_completion_bash.md) - Generate the autocompletion script for bash
- [`suprsend completion fish`](references/suprsend_completion_fish.md) - Generate the autocompletion script for fish
- [`suprsend completion powershell`](references/suprsend_completion_powershell.md) - Generate the autocompletion script for powershell
- [`suprsend completion zsh`](references/suprsend_completion_zsh.md) - Generate the autocompletion script for zsh
- [`suprsend event`](references/suprsend_event.md) - Manage events
- [`suprsend event get`](references/suprsend_event_get.md) - Get events
- [`suprsend event list`](references/suprsend_event_list.md) - List events
- [`suprsend event pull`](references/suprsend_event_pull.md) - Pull events from workspace to local directory
- [`suprsend event push`](references/suprsend_event_push.md) - Push linked events
- [`suprsend generate-types`](references/suprsend_generate-types.md) - Generate type definitions from JSON Schema
- [`suprsend generate-types dart`](references/suprsend_generate-types_dart.md) - Generate Dart types from JSON Schema
- [`suprsend generate-types go`](references/suprsend_generate-types_go.md) - Generate Go types from JSON Schema
- [`suprsend generate-types java`](references/suprsend_generate-types_java.md) - Generate Java types from JSON Schema
- [`suprsend generate-types kotlin`](references/suprsend_generate-types_kotlin.md) - Generate Kotlin types from JSON Schema
- [`suprsend generate-types python`](references/suprsend_generate-types_python.md) - Generate Python types from JSON Schema
- [`suprsend generate-types swift`](references/suprsend_generate-types_swift.md) - Generate Swift types from JSON Schema
- [`suprsend generate-types typescript`](references/suprsend_generate-types_typescript.md) - Generate TypeScript types from JSON Schema
- [`suprsend genskills`](references/suprsend_genskills.md) - Generate SKILLS.md
- [`suprsend profile`](references/suprsend_profile.md) - Manage Profile
- [`suprsend profile add`](references/suprsend_profile_add.md) - Add a new profile
- [`suprsend profile list`](references/suprsend_profile_list.md) - List all profiles
- [`suprsend profile modify`](references/suprsend_profile_modify.md) - Modify a profile
- [`suprsend profile remove`](references/suprsend_profile_remove.md) - Remove a profile
- [`suprsend profile use`](references/suprsend_profile_use.md) - Set the active profile
- [`suprsend schema`](references/suprsend_schema.md) - Manage trigger payload schemas
- [`suprsend schema commit`](references/suprsend_schema_commit.md) - Commit schema from draft to live
- [`suprsend schema get`](references/suprsend_schema_get.md) - Get schema details
- [`suprsend schema list`](references/suprsend_schema_list.md) - List schemas
- [`suprsend schema pull`](references/suprsend_schema_pull.md) - Pull schemas
- [`suprsend schema push`](references/suprsend_schema_push.md) - Push schemas
- [`suprsend start-mcp-server`](references/suprsend_start-mcp-server.md) - Start SuprSend MCP server
- [`suprsend start-mcp-server list-tools`](references/suprsend_start-mcp-server_list-tools.md) - List all the tools supported by the server
- [`suprsend sync`](references/suprsend_sync.md) - Sync SuprSend assets from one workspace to another
- [`suprsend translation`](references/suprsend_translation.md) - Manage Translations
- [`suprsend translation commit`](references/suprsend_translation_commit.md) - Commit translation
- [`suprsend translation get`](references/suprsend_translation_get.md) - Get translations
- [`suprsend translation list`](references/suprsend_translation_list.md) - List Translations
- [`suprsend translation pull`](references/suprsend_translation_pull.md) - Pull Translation files
- [`suprsend translation push`](references/suprsend_translation_push.md) - Push translation files to a workspace
- [`suprsend workflow`](references/suprsend_workflow.md) - Manage workflows
- [`suprsend workflow disable`](references/suprsend_workflow_disable.md) - Disable a workflow
- [`suprsend workflow enable`](references/suprsend_workflow_enable.md) - Enables a workflow.
- [`suprsend workflow get`](references/suprsend_workflow_get.md) - Get workflow details
- [`suprsend workflow list`](references/suprsend_workflow_list.md) - List workflows for a workspace
- [`suprsend workflow pull`](references/suprsend_workflow_pull.md) - Pull workflows from SuprSend workspace to local
- [`suprsend workflow push`](references/suprsend_workflow_push.md) - Push workflows from local to SuprSend workspace

See [references/suprsend.md](references/suprsend.md) for root command flags.

Run `suprsend --help` or `suprsend <command> --help` for full usage details.
