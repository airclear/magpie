package gui

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yetone/magpie/internal/prompts"
	"github.com/yetone/magpie/internal/settings"
)

var errPromptInsert = errors.New("automatic insertion is not available on this platform")

// promptApp is the application that was in front when the picker opened.
// Its fields mean what the platform's insertion code makes them mean.
type promptApp struct {
	ID   string
	Name string
}

// promptInsertResult says what happened after the picker chose a prompt.
type promptInsertResult struct {
	Inserted    bool   `json:"inserted"`
	Copied      bool   `json:"copied"`
	NeedsAccess bool   `json:"needsAccess,omitempty"`
	Error       string `json:"error,omitempty"`
}

type promptHost interface {
	PromptShortcut() (string, string)
	PromptCanInsert() bool
	PromptNeedsAccess() bool
	ShowPrompt()
	HidePrompt()
	InsertPrompt(text string) promptInsertResult
	OpenPromptAccessibilitySettings()
}

// promptShortcutChanged registers a new shortcut before its setting is saved.
var promptShortcutChanged func(string) error

type promptListJSON struct {
	Prompts         []prompts.Prompt `json:"prompts"`
	Shortcut        string           `json:"shortcut"`
	ShortcutError   string           `json:"shortcutError,omitempty"`
	InsertAvailable bool             `json:"insertAvailable"`
	NeedsAccess     bool             `json:"needsAccess,omitempty"`
}

func promptRoutes(mux *http.ServeMux, w Windows) {
	host, _ := w.(promptHost)

	mux.HandleFunc("GET /api/prompts", func(rw http.ResponseWriter, r *http.Request) {
		list, err := prompts.List()
		if err != nil {
			fail(rw, err)
			return
		}
		out := promptListJSON{Prompts: list, Shortcut: settings.Load().PromptShortcut}
		if host != nil {
			if shortcut, why := host.PromptShortcut(); shortcut != "" {
				out.Shortcut = shortcut
				out.ShortcutError = why
			} else {
				out.ShortcutError = why
			}
			out.InsertAvailable = host.PromptCanInsert()
			out.NeedsAccess = host.PromptNeedsAccess()
		}
		writeJSON(rw, out)
	})
	mux.HandleFunc("POST /api/prompts/save", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		if _, err := prompts.Save(in.ID, in.Name, in.Text); err != nil {
			fail(rw, err)
			return
		}
		promptJSON(rw, host)
	})
	mux.HandleFunc("POST /api/prompts/remove", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		if err := prompts.Remove(in.ID); err != nil {
			fail(rw, err)
			return
		}
		promptJSON(rw, host)
	})
	mux.HandleFunc("POST /api/prompts/insert", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		p, err := prompts.Find(in.ID)
		if err != nil {
			fail(rw, err)
			return
		}
		if host == nil {
			if w != nil && w.Copy(p.Text) {
				writeJSON(rw, promptInsertResult{Copied: true, Error: "Automatic insertion is not available in this build. The prompt is on the clipboard."})
				return
			}
			fail(rw, errors.New("automatic insertion is not available in this build"))
			return
		}
		writeJSON(rw, host.InsertPrompt(p.Text))
	})
	mux.HandleFunc("POST /api/prompts/show", func(rw http.ResponseWriter, r *http.Request) {
		if host == nil {
			fail(rw, errors.New("the prompt picker is not available here"))
			return
		}
		host.ShowPrompt()
		writeJSON(rw, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /api/prompts/hide", func(rw http.ResponseWriter, r *http.Request) {
		if host != nil {
			host.HidePrompt()
		}
		writeJSON(rw, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /api/prompts/accessibility", func(rw http.ResponseWriter, r *http.Request) {
		if host == nil {
			fail(rw, errors.New("accessibility settings are not available here"))
			return
		}
		host.OpenPromptAccessibilitySettings()
		writeJSON(rw, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /api/settings/prompt-shortcut", func(rw http.ResponseWriter, r *http.Request) {
		var in struct {
			Shortcut string `json:"shortcut"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			fail(rw, err)
			return
		}
		shortcut := strings.TrimSpace(in.Shortcut)
		if shortcut == "" {
			fail(rw, errors.New("the prompt shortcut cannot be empty"))
			return
		}
		cur, err := settings.LoadChecked()
		if err != nil {
			fail(rw, err)
			return
		}
		old := cur.PromptShortcut
		if shortcut == old {
			writeJSON(rw, settingsState())
			return
		}
		if promptShortcutChanged != nil {
			if err := promptShortcutChanged(shortcut); err != nil {
				fail(rw, err)
				return
			}
		}
		cur.PromptShortcut = shortcut
		if err := settings.Save(cur); err != nil {
			if promptShortcutChanged != nil {
				_ = promptShortcutChanged(old)
			}
			fail(rw, err)
			return
		}
		writeJSON(rw, settingsState())
	})
}

func promptJSON(rw http.ResponseWriter, host promptHost) {
	list, err := prompts.List()
	if err != nil {
		fail(rw, err)
		return
	}
	out := promptListJSON{Prompts: list, Shortcut: settings.Load().PromptShortcut}
	if host != nil {
		if shortcut, why := host.PromptShortcut(); shortcut != "" {
			out.Shortcut = shortcut
			out.ShortcutError = why
		} else {
			out.ShortcutError = why
		}
		out.InsertAvailable = host.PromptCanInsert()
		out.NeedsAccess = host.PromptNeedsAccess()
	}
	writeJSON(rw, out)
}
