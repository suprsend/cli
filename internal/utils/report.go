package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/suprsend/cli/internal/config"
	"github.com/suprsend/cli/internal/termio"
	"github.com/suprsend/cli/mgmnt"
)

// Item statuses used in a Report.
const (
	StatusPushed      = "pushed"       // saved as draft
	StatusCommitted   = "committed"    // promoted to live
	StatusUnchanged   = "unchanged"    // commit requested, but the draft already matched live
	StatusPulled      = "pulled"       // written to a local file
	StatusWouldPush   = "would-push"   // --dry-run
	StatusWouldCommit = "would-commit" // --dry-run
	StatusEnabled     = "enabled"
	StatusDisabled    = "disabled"
	StatusTriggered   = "triggered"
	StatusGenerated   = "generated"
	StatusSkipped     = "skipped"
	StatusFailed      = "failed"
)

// ReportItem is the outcome for one resource.
type ReportItem struct {
	Resource string `json:"resource" yaml:"resource"`
	Status   string `json:"status" yaml:"status"`
	Detail   string `json:"detail,omitempty" yaml:"detail,omitempty"`
	Error    string `json:"error,omitempty" yaml:"error,omitempty"`
}

// Report is the result of a command that pushes, pulls, commits or otherwise
// changes resources. Results always go to stdout, whether or not it is a
// terminal: in pretty mode one line per item as it happens plus a summary,
// and with -o json or -o yaml one document from Finish. Progress and log
// lines stay on stderr.
type Report struct {
	Command   string         `json:"command" yaml:"command"`
	From      string         `json:"from_workspace,omitempty" yaml:"from_workspace,omitempty"`
	Workspace string         `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Host      string         `json:"host,omitempty" yaml:"host,omitempty"`
	Profile   string         `json:"profile,omitempty" yaml:"profile,omitempty"`
	DryRun    bool           `json:"dry_run" yaml:"dry_run"`
	OK        bool           `json:"ok" yaml:"ok"`
	Counts    map[string]int `json:"counts" yaml:"counts"`
	Items     []ReportItem   `json:"items" yaml:"items"`
	Warnings  []string       `json:"warnings,omitempty" yaml:"warnings,omitempty"`

	order []string // statuses in the order first seen, for the pretty summary
}

// NewReport starts a report and, in pretty mode, prints which workspace and
// host the command acts on.
func NewReport(command, workspace string, dryRun bool) *Report {
	r := newReport(command, workspace, dryRun)
	target := fmt.Sprintf("workspace %q", workspace)
	if workspace == "" {
		target = "local files"
	}
	r.printHeader(target)
	return r
}

// NewSyncReport starts a report for a command that reads from one workspace
// and writes to another.
func NewSyncReport(command, from, to string, dryRun bool) *Report {
	r := newReport(command, to, dryRun)
	r.From = from
	r.printHeader(fmt.Sprintf("from workspace %q to workspace %q", from, to))
	return r
}

func newReport(command, workspace string, dryRun bool) *Report {
	return &Report{
		Command:   command,
		Workspace: workspace,
		Host:      hostOf(config.Cfg.MgmntUrl.Value),
		Profile:   config.Cfg.Profile.Value,
		DryRun:    dryRun,
		Counts:    map[string]int{},
		Items:     []ReportItem{},
	}
}

func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

func (r *Report) pretty() bool {
	return config.Cfg.OutputType.Value != "json" && config.Cfg.OutputType.Value != "yaml" && !config.Cfg.Quiet.Value
}

func (r *Report) println(s string) {
	fmt.Fprintln(termio.LogWriter(os.Stdout), s)
}

func (r *Report) printHeader(target string) {
	if !r.pretty() {
		return
	}
	line := r.Command
	if r.DryRun {
		line += " (dry run)"
	}
	line += ": " + target
	if r.Host != "" {
		line += " on " + r.Host
	}
	if r.Profile != "" {
		line += fmt.Sprintf(" (profile %q)", r.Profile)
	}
	r.println(line)
}

// Add records the outcome for one resource, e.g. Add("templates/welcome", StatusCommitted, "").
func (r *Report) Add(resource, status, detail string) {
	r.add(ReportItem{Resource: resource, Status: status, Detail: detail})
}

// Fail records a failed resource. The pretty line names the resource only:
// the reason is printed once, in the error the command returns.
func (r *Report) Fail(resource string, err error) {
	item := ReportItem{Resource: resource, Status: StatusFailed}
	if err != nil {
		item.Error = err.Error()
	}
	r.add(item)
}

func (r *Report) add(item ReportItem) {
	r.Items = append(r.Items, item)
	if r.Counts[item.Status] == 0 {
		r.order = append(r.order, item.Status)
	}
	r.Counts[item.Status]++
	if !r.pretty() {
		return
	}
	line := fmt.Sprintf("%-12s %s", statusLabel(item.Status), item.Resource)
	if item.Detail != "" {
		line += " (" + item.Detail + ")"
	}
	r.println(line)
}

func statusLabel(status string) string {
	label := fmt.Sprintf("%-12s", status)
	switch status {
	case StatusFailed:
		return color.RedString(label)
	case StatusCommitted, StatusPushed, StatusPulled, StatusEnabled, StatusDisabled, StatusTriggered, StatusGenerated:
		return color.GreenString(label)
	case StatusSkipped:
		return color.YellowString(label)
	}
	return label
}

// Count returns how many items have the given status.
func (r *Report) Count(status string) int { return r.Counts[status] }

// Finish prints the summary (pretty) or the whole report (json/yaml) and
// returns err unchanged, so a command can end with `return report.Finish(err)`.
// A failed command that recorded no items prints nothing: the error says it all.
func (r *Report) Finish(err error) error {
	r.OK = err == nil
	r.Warnings = append(r.Warnings, config.HiddenWarnings()...)
	if err != nil && len(r.Items) == 0 {
		return err
	}
	switch config.Cfg.OutputType.Value {
	case "json", "yaml":
		OutputData(r, config.Cfg.OutputType.Value)
		return err
	}
	if config.Cfg.Quiet.Value {
		return err
	}
	for _, w := range r.Warnings {
		r.println("warning: " + w)
	}
	r.println(r.summary())
	return err
}

func (r *Report) summary() string {
	if len(r.Items) == 0 {
		return "Nothing to do"
	}
	parts := make([]string, 0, len(r.order))
	for _, status := range r.order {
		parts = append(parts, fmt.Sprintf("%d %s", r.Counts[status], status))
	}
	s := "Done: " + strings.Join(parts, ", ")
	if r.DryRun {
		s += " (dry run: nothing was changed)"
	}
	return s
}

// PushStatus maps the commit status of a push (or commit) request to a report
// status and detail: pushed (draft only), committed, or unchanged.
func PushStatus(s mgmnt.CommitStatus) (string, string) {
	switch s {
	case mgmnt.Committed:
		return StatusCommitted, ""
	case mgmnt.NoChanges:
		return StatusUnchanged, "no changes to commit"
	}
	return StatusPushed, ""
}
