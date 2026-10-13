/*
Copyright © 2025 SuprSend
*/
package template

import (
	"github.com/spf13/cobra"
)

// templateLayoutHelp documents the on-disk layout written by `template pull`
// and read by `template push` (see writeOneTemplate / readTemplateVariants).
const templateLayoutHelp = "Local layout: each template lives in `<dir>/<slug>/` (default dir `./suprsend/templates`):\n\n" +
	"- `template.json`: `slug`, `name`, `enabled_channels`, and `variant_order` (variant ids in order, keyed by `<channel>` or `<channel>/<tenant_id>` for tenant-specific ordering). Push sends `enabled_channels` and `variant_order`; `name` is informational and is not pushed.\n" +
	"- `mock_data.json`: mock data used for previews (optional).\n" +
	"- `<channel>/<variant_id>/variant.json`: one directory per variant.\n" +
	"- `<channel>/_tenants/<tenant_id>/<variant_id>/variant.json`: tenant-specific variants.\n" +
	"- Body files next to `variant.json`, referenced from it as `\"@<file>\"`: `body.raw.html`, `body.raw.txt`, `body.designer.json`, `body.designer.html`, `body.designer.txt`, `body.plain_text.txt` (email), `body.block.jsonnet` (Slack / MS Teams block kit), `body.card.jsonnet` (MS Teams jsonnet card), `body.md` (inbox). Push inlines them back into the variant.\n\n" +
	"The variant's channel, id, and tenant are read from the fields inside `variant.json`; the directory names only have to follow the shape above."

// TemplateCmd represents the template command
var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `Manage notification templates. Templates define the content and structure of notifications across channels (email, SMS, push, in-app, etc.). Subcommands let you list, get details, pull to local files, push from local files, and commit templates.`,
	Example: `  suprsend template list
  suprsend template get welcome-email
  suprsend template pull --dir ./suprsend/templates
  suprsend template push welcome-email --commit`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
