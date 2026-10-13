package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestGenerateSkillsRemovesStaleReferences(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, skillsConfig.Name, "references")
	stale := filepath.Join(refDir, "suprsend_category_get.md")
	writeFile(t, stale, "# a command that no longer exists\n")

	if err := generateSkills(dir); err != nil {
		t.Fatalf("generateSkills: %v", err)
	}

	if exists(stale) {
		t.Errorf("stale reference %s was not removed", stale)
	}
	for _, name := range []string{"suprsend.md", "suprsend_sync.md", "suprsend_version.md"} {
		if !exists(filepath.Join(refDir, name)) {
			t.Errorf("expected reference %s to be generated", name)
		}
	}
	for _, name := range []string{"suprsend_genskills.md", "suprsend_gendocs.md"} {
		if exists(filepath.Join(refDir, name)) {
			t.Errorf("dev-only command page %s should not be generated", name)
		}
	}

	skill, err := os.ReadFile(filepath.Join(dir, skillsConfig.Name, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skill), "`suprsend version`") {
		t.Error("SKILL.md should list `suprsend version`")
	}
	if strings.Contains(string(skill), "genskills") || strings.Contains(string(skill), "gendocs") {
		t.Error("SKILL.md should not list dev-only generator commands")
	}
}

func TestGenerateDocsRemovesStalePages(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "suprsend_category_get.md")
	unrelated := filepath.Join(dir, "README.md")
	writeFile(t, stale, "## a command that no longer exists\n")
	writeFile(t, unrelated, "not generated\n")

	if err := generateDocs(dir); err != nil {
		t.Fatalf("generateDocs: %v", err)
	}

	if exists(stale) {
		t.Errorf("stale page %s was not removed", stale)
	}
	if !exists(unrelated) {
		t.Errorf("non-generated file %s should be kept", unrelated)
	}
	for _, name := range []string{"suprsend.md", "suprsend_sync.md", "suprsend_version.md"} {
		if !exists(filepath.Join(dir, name)) {
			t.Errorf("expected page %s to be generated", name)
		}
	}
	if exists(filepath.Join(dir, "suprsend_genskills.md")) {
		t.Error("dev-only genskills page should not be generated")
	}
}

// The version command's library-provided example embeds os.Args[0]; the
// generated docs must not depend on how the binary was invoked.
func TestVersionExampleIsStable(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "version" {
			continue
		}
		if strings.Contains(c.Example, os.Args[0]) {
			t.Errorf("version example contains the binary path %q", os.Args[0])
		}
		if !strings.Contains(c.Example, "suprsend version") {
			t.Errorf("version example = %q, want it to use `suprsend version`", c.Example)
		}
		return
	}
	t.Fatal("version command not registered")
}
