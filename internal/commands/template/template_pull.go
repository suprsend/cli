/*
Copyright © 2025 SuprSend
*/
package template

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/suprsend/cli/internal/clierr"
	"github.com/suprsend/cli/internal/utils"
	"github.com/suprsend/cli/mgmnt"
	"golang.org/x/sync/errgroup"
)

// templateFetchConcurrency caps in-flight template detail fetches when pulling
// the full workspace. Each template requires three sequential GETs (variants,
// mock data, variant order); fanning out across templates is the easy win.
const templateFetchConcurrency = 20

type TemplateResult struct {
	Slug            string                      `json:"slug"`
	Name            string                      `json:"name"`
	EnabledChannels []string                    `json:"enabled_channels"`
	Variants        []map[string]any            `json:"variants"`
	MockData        map[string]any              `json:"mock_data,omitempty"`
	VariantOrder    *mgmnt.VariantOrderResponse `json:"variant_order,omitempty"`
}

// FetchOptions configures optional callbacks for FetchTemplates. Both fields
// are safe to leave nil — sync.go does — in which case FetchTemplates buffers
// every template into the returned slice with no incremental signal.
//
// OnListed fires exactly once after the workspace listing completes,
// before any per-template fetch starts. The argument is the total template
// count and lets streaming consumers prepare progress UI.
//
// OnResult fires once per successfully-fetched template, from the
// concurrent fetch goroutines. Implementations must be goroutine-safe —
// guard any shared state (file writes are typically safe per-template,
// shared counters need an atomic).
type FetchOptions struct {
	OnListed func(total int)
	OnResult func(TemplateResult)
}

// FetchTemplates fetches either one template (when slug != "") or all templates
// from the workspace, returning the assembled results ready to hand to
// WriteTemplatesToFiles. Per-template fetch errors when slug == "" are logged
// and the offending template is skipped; only failures while listing templates
// or fetching a specifically requested slug are returned.
//
// opts is optional; pass FetchOptions{} when no streaming signal is needed.
func FetchTemplates(ctx context.Context, client *mgmnt.SS_MgmntClient, workspace, mode, slug string, opts FetchOptions) ([]TemplateResult, error) {
	var results []TemplateResult

	if slug != "" {
		tmpl, err := client.GetTemplate(ctx, workspace, slug, mode)
		if err != nil {
			return nil, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch template %s", slug))
		}
		variants, err := client.GetTemplateVariants(ctx, workspace, slug, mode)
		if err != nil {
			return nil, clierr.Wrap(err, clierr.CodeAPIInternal, fmt.Sprintf("couldn't fetch variants for template %s", slug))
		}
		mockData, err := client.GetTemplateMockData(ctx, workspace, slug)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", slug)
		}
		variantOrder, err := client.GetVariantOrder(ctx, workspace, slug, mode)
		if err != nil {
			log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", slug)
		}
		r := TemplateResult{
			Slug:            slug,
			Name:            tmpl.Name,
			EnabledChannels: tmpl.EnabledChannels,
			Variants:        variants,
			MockData:        mockData,
			VariantOrder:    variantOrder,
		}
		if opts.OnListed != nil {
			opts.OnListed(1)
		}
		if opts.OnResult != nil {
			opts.OnResult(r)
		}
		results = append(results, r)
		return results, nil
	}

	templates, err := client.ListTemplates(ctx, workspace, math.MaxInt32, 0, mode)
	if err != nil {
		return nil, clierr.Wrap(err, clierr.CodeAPIInternal, "couldn't fetch templates")
	}
	if opts.OnListed != nil {
		opts.OnListed(len(templates.Results))
	}

	// Per-template fetch is three sequential GETs; for workspaces with many
	// templates the serial form takes minutes. Parallelize across templates
	// with bounded concurrency. A failed variants fetch skips that template
	// (slot stays nil); mock-data / variant-order failures only log.
	slots := make([]*TemplateResult, len(templates.Results))
	g := new(errgroup.Group)
	g.SetLimit(templateFetchConcurrency)
	for i, t := range templates.Results {
		i, t := i, t
		g.Go(func() error {
			variants, err := client.GetTemplateVariants(ctx, workspace, t.Slug, mode)
			if err != nil {
				log.WithError(err).Errorf("Couldn't fetch variants for template: %s", t.Slug)
				return nil
			}
			mockData, err := client.GetTemplateMockData(ctx, workspace, t.Slug)
			if err != nil {
				log.WithError(err).Warnf("Couldn't fetch mock data for template: %s", t.Slug)
			}
			variantOrder, err := client.GetVariantOrder(ctx, workspace, t.Slug, mode)
			if err != nil {
				log.WithError(err).Warnf("Couldn't fetch variant order for template: %s", t.Slug)
			}
			r := TemplateResult{
				Slug:            t.Slug,
				Name:            t.Name,
				EnabledChannels: t.EnabledChannels,
				Variants:        variants,
				MockData:        mockData,
				VariantOrder:    variantOrder,
			}
			slots[i] = &r
			if opts.OnResult != nil {
				opts.OnResult(r)
			}
			return nil
		})
	}
	_ = g.Wait()

	for _, s := range slots {
		if s != nil {
			results = append(results, *s)
		}
	}
	return results, nil
}

var templatePullCmd = &cobra.Command{
	Use:   "pull [<slug>]",
	Short: "Pull templates and their variants from SuprSend workspace",
	Long:  `Pull templates and their variants from SuprSend workspace. Pass a slug as a positional argument or via --slug to pull a single template, or omit to pull all.`,
	Example: `  # Pull all templates to default directory (suprsend/templates/)
  suprsend template pull

  # Pull a single template by slug
  suprsend template pull welcome-email

  # Pull to a custom directory using the flag form
  suprsend template pull --slug welcome-email --dir ./my-templates`,
	Annotations: map[string]string{
		"skills:tip.a-overwrite": "Pull overwrites local template files for the matched slugs. Commit local edits first if you don't want them clobbered (or use `--force` to skip the prompt).",
		"skills:tip.b-mode":      "Defaults to the **live** mode. Use `--mode draft` to mirror the pending state instead.",
	},
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _ := cmd.Flags().GetString("workspace")
		mode, _ := cmd.Flags().GetString("mode")
		slug := utils.ResolveSlug(cmd, args)
		outputDir, _ := cmd.Flags().GetString("dir")
		force, _ := cmd.Flags().GetBool("force")

		// Directory resolution
		if outputDir == "" {
			outputDir = filepath.Join(".", "suprsend", "templates")
			if _, err := os.Stat(outputDir); os.IsNotExist(err) {
				if force {
					log.Infof("Using default directory: %s", outputDir)
				} else {
					od, success := promptForOutputDirectory()
					if !success {
						return clierr.New("output directory is required; use --dir to specify one", clierr.CodeInvalidUsage)
					}
					outputDir = od
				}
			}
			if outputDir == "" {
				return clierr.New("no output directory specified; use --dir to set one", clierr.CodeInvalidUsage)
			}
		}

		if err := ensureOutputDirectory(outputDir); err != nil {
			return err
		}
		if err := EnsureTemplatesOutputDir(outputDir); err != nil {
			return err
		}

		spinner := utils.NewSpinner("Listing templates...")

		mgmntClient := utils.GetSuprSendMgmntClient()

		// Stream each template to disk as it finishes fetching, with a live
		// progress counter on the spinner. statsMu guards stats since the
		// fetch goroutines all share it.
		stats := &TemplateWriteStats{Errors: []string{}}
		var statsMu sync.Mutex
		var done atomic.Int32
		var total atomic.Int32
		var totalVariants atomic.Int32

		results, fetchErr := FetchTemplates(cmd.Context(), mgmntClient, workspace, mode, slug, FetchOptions{
			OnListed: func(n int) {
				total.Store(int32(n))
				stats.Total = n
				spinner.UpdateMessage(fmt.Sprintf("Pulled 0/%d templates", n))
			},
			OnResult: func(t TemplateResult) {
				statsMu.Lock()
				_ = WriteOneTemplate(t, outputDir, stats)
				statsMu.Unlock()
				totalVariants.Add(int32(len(t.Variants)))
				n := done.Add(1)
				spinner.UpdateMessage(fmt.Sprintf("Pulled %d/%d templates", n, total.Load()))
			},
		})
		if fetchErr != nil {
			spinner.Stop("Failed")
			return fetchErr
		}

		spinner.Stop(fmt.Sprintf("Pulled %d templates with %d variants from %s", len(results), totalVariants.Load(), workspace))

		log.Infof("Pull Summary: %d total, %d success, %d failed", stats.Total, stats.Success, stats.Failed)
		if len(stats.Errors) > 0 {
			log.Info("Errors:")
			for _, e := range stats.Errors {
				log.Infof("  - %s", e)
			}
		}
		return nil
	},
}

func init() {
	templatePullCmd.PersistentFlags().StringP("mode", "m", "live", "Version mode: draft or live")
	templatePullCmd.PersistentFlags().StringP("slug", "g", "", "Slug of a specific template to pull")
	templatePullCmd.PersistentFlags().StringP("dir", "d", "", "Output directory for templates (default: ./suprsend/templates)")
	templatePullCmd.PersistentFlags().BoolP("force", "F", false, "Force using default directory without prompting")
	TemplateCmd.AddCommand(templatePullCmd)
}
