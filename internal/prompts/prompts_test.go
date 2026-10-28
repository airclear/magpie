package prompts

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPromptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	first, err := Save("", "Review", "Review this change for bugs.")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Save("", "Explain", "Explain this code simply.")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.ID == second.ID {
		t.Fatalf("ids: %q %q", first.ID, second.ID)
	}

	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Review" || got[1].Name != "Explain" {
		t.Fatalf("saved: %+v", got)
	}
	if runtime.GOOS != "windows" {
		st, err := os.Stat(Path())
		if err != nil {
			t.Fatal(err)
		}
		if mode := st.Mode().Perm(); mode != 0o600 {
			t.Fatalf("prompts.json mode %o, want 600", mode)
		}
	}

	first.Name, first.Text = "  Code review  ", "Find bugs and missing tests."
	if _, err := Save(first.ID, first.Name, first.Text); err != nil {
		t.Fatal(err)
	}
	got, err = List()
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "Code review" || got[0].Text != "Find bugs and missing tests." {
		t.Fatalf("changed: %+v", got[0])
	}

	if err := Remove(second.ID); err != nil {
		t.Fatal(err)
	}
	got, err = List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != first.ID {
		t.Fatalf("removed: %+v", got)
	}
}

func TestPromptNeedsNameAndText(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Save("", "", "text"); err == nil {
		t.Fatal("a prompt without a name was saved")
	}
	if _, err := Save("", "name", " \n"); err == nil {
		t.Fatal("a prompt without text was saved")
	}
}

func TestBrokenPromptFileIsNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	broken := []byte(`{"prompts":[`)
	if err := os.WriteFile(Path(), broken, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "prompts.json") {
		t.Fatalf("List error: %v", err)
	}
	if _, err := Save("", "Review", "text"); err == nil {
		t.Fatal("Save overwrote a broken file")
	}
	got, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(broken) {
		t.Fatalf("file changed to %q", got)
	}
}

func TestMissingPromptIsNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := Find("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find: %v", err)
	}
	if err := Remove("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Remove: %v", err)
	}
}
