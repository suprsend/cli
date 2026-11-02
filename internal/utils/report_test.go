package utils

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/suprsend/cli/internal/config"
)

// captureStdout runs f and returns what it wrote to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	f()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func withOutputType(t *testing.T, outputType string) {
	t.Helper()
	prev := *config.Cfg
	config.Cfg.OutputType = config.ConfigString{Value: outputType}
	config.Cfg.MgmntUrl = config.ConfigString{Value: "https://management-api.suprsend.com/"}
	config.Cfg.Profile = config.ConfigString{Value: "staging"}
	t.Cleanup(func() { *config.Cfg = prev })
}

// Results go to stdout even when it is not a terminal (here, a pipe).
func TestReportPrettyWritesToPipedStdout(t *testing.T) {
	withOutputType(t, "pretty")
	out := captureStdout(t, func() {
		r := NewReport("template push", "evals-staging", false)
		r.Add("templates/welcome", StatusCommitted, "")
		r.Add("templates/reset", StatusUnchanged, "no changes to commit")
		r.Fail("templates/broken", errors.New("boom"))
		_ = r.Finish(errors.New("1 template(s) failed to push"))
	})
	for _, want := range []string{
		`template push: workspace "evals-staging" on management-api.suprsend.com (profile "staging")`,
		"committed    templates/welcome",
		"unchanged    templates/reset (no changes to commit)",
		"failed       templates/broken",
		"Done: 1 committed, 1 unchanged, 1 failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}
	// The failure reason is printed once, by the error, not in the report.
	if strings.Contains(out, "boom") {
		t.Errorf("pretty output repeats the failure reason:\n%s", out)
	}
}

func TestReportJSON(t *testing.T) {
	withOutputType(t, "json")
	out := captureStdout(t, func() {
		r := NewReport("workflow push", "staging", true)
		r.Add("workflows/welcome", StatusWouldCommit, "")
		_ = r.Finish(nil)
	})
	var got Report
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, out)
	}
	if !got.OK || !got.DryRun || got.Workspace != "staging" || got.Counts[StatusWouldCommit] != 1 || len(got.Items) != 1 {
		t.Errorf("unexpected report: %+v", got)
	}
}

// A command that fails before doing anything prints no report; the error
// alone describes it.
func TestReportFinishWithoutItems(t *testing.T) {
	withOutputType(t, "json")
	out := captureStdout(t, func() {
		_ = NewReport("event pull", "staging", false).Finish(errors.New("resource not found"))
	})
	if out != "" {
		t.Errorf("expected no stdout, got %q", out)
	}
}

func TestClosestSlug(t *testing.T) {
	slugs := []string{"staging", "production", "evals-staging", "evals-pool-1"}
	if got := closestSlug("evals-stagign", slugs); got != "evals-staging" {
		t.Errorf("closestSlug(evals-stagign) = %q, want evals-staging", got)
	}
	if got := closestSlug("totally-different-name", slugs); got != "" {
		t.Errorf("closestSlug(totally-different-name) = %q, want no suggestion", got)
	}
}
