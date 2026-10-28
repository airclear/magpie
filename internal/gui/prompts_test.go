package gui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/prompts"
	"github.com/yetone/magpie/internal/testenv"
)

type promptTestHost struct {
	shortcut string
	err      string
	inserted string
}

func (h *promptTestHost) PromptShortcut() (string, string) { return h.shortcut, h.err }
func (h *promptTestHost) PromptCanInsert() bool            { return true }
func (h *promptTestHost) PromptNeedsAccess() bool          { return false }
func (h *promptTestHost) ShowPrompt()                      {}
func (h *promptTestHost) HidePrompt()                      {}
func (h *promptTestHost) OpenPromptAccessibilitySettings() {}
func (h *promptTestHost) InsertPrompt(text string) promptInsertResult {
	h.inserted = text
	return promptInsertResult{Inserted: true}
}

type promptTestWindows struct {
	Windows
	*promptTestHost
}

func (w promptTestWindows) Copy(text string) bool {
	w.inserted = "copied:" + text
	return true
}

func TestPromptsAPI(t *testing.T) {
	dir := t.TempDir()
	testenv.SetHome(t, dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	host := &promptTestHost{shortcut: "CmdOrCtrl+Shift+Space"}
	mux := http.NewServeMux()
	promptRoutes(mux, promptTestWindows{promptTestHost: host})
	call := func(method, path string, body any, out any) int {
		t.Helper()
		var r *http.Request
		if body == nil {
			r = httptest.NewRequest(method, path, nil)
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			r = httptest.NewRequest(method, path, bytes.NewReader(b))
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if out != nil && w.Code == http.StatusOK {
			if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
				t.Fatalf("%s: %v\n%s", path, err, w.Body)
			}
		}
		return w.Code
	}

	var list promptListJSON
	if code := call("GET", "/api/prompts", nil, &list); code != http.StatusOK || len(list.Prompts) != 0 || list.Shortcut != host.shortcut {
		t.Fatalf("empty list: %d %+v", code, list)
	}
	var saved promptListJSON
	if code := call("POST", "/api/prompts/save", map[string]string{"name": "Review", "text": "Find bugs."}, &saved); code != http.StatusOK || len(saved.Prompts) != 1 {
		t.Fatalf("save: %d %+v", code, saved)
	}
	id := saved.Prompts[0].ID
	if code := call("POST", "/api/prompts/insert", map[string]string{"id": id}, &promptInsertResult{}); code != http.StatusOK {
		t.Fatalf("insert: %d", code)
	}
	if host.inserted != "Find bugs." {
		t.Fatalf("inserted %q", host.inserted)
	}
	if code := call("POST", "/api/prompts/remove", map[string]string{"id": id}, &saved); code != http.StatusOK || len(saved.Prompts) != 0 {
		t.Fatalf("remove: %d %+v", code, saved)
	}
	if got, err := prompts.List(); err != nil || len(got) != 0 {
		t.Fatalf("stored: %+v %v", got, err)
	}
}

func TestPromptShortcutSetting(t *testing.T) {
	dir := t.TempDir()
	testenv.SetHome(t, dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	var got string
	old := promptShortcutChanged
	promptShortcutChanged = func(s string) error { got = s; return nil }
	t.Cleanup(func() { promptShortcutChanged = old })

	mux := http.NewServeMux()
	promptRoutes(mux, nil)
	r := httptest.NewRequest("POST", "/api/settings/prompt-shortcut", bytes.NewReader([]byte(`{"shortcut":"CmdOrCtrl+Shift+P"}`)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || got != "CmdOrCtrl+Shift+P" {
		t.Fatalf("%d %s, registered %q", w.Code, w.Body, got)
	}
}
