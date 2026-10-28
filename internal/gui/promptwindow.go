//go:build !nogui

package gui

import (
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func promptOptions(goos, query string) application.WebviewWindowOptions {
	url := "/prompt.html"
	if query != "" {
		url += "?" + strings.TrimPrefix(query, "&")
	}
	o := application.WebviewWindowOptions{
		Name:            "prompt-picker",
		Title:           "Prompts",
		URL:             url,
		Width:           680,
		Height:          430,
		MinWidth:        420,
		MinHeight:       220,
		MaxWidth:        900,
		MaxHeight:       620,
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		BackgroundType:  application.BackgroundTypeTranslucent,
		Mac: application.MacWindow{
			Backdrop:     application.MacBackdropTranslucent,
			CornerRadius: 18,
		},
		Windows: application.WindowsWindow{HiddenOnTaskbar: true},
	}
	if goos == "linux" {
		o.BackgroundType = application.BackgroundTypeSolid
	}
	return o
}

func (h *host) makePrompt() *application.WebviewWindow {
	w := h.app.Window.NewWithOptions(promptOptions(runtime.GOOS, h.query))
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		w.Hide()
	})
	return w
}

// changePromptShortcut moves the global shortcut without leaving the old one
// behind when the new one is invalid or another application owns it.
func (h *host) changePromptShortcut(next string) error {
	h.promptMu.Lock()
	defer h.promptMu.Unlock()
	if next == h.promptRegistered {
		h.promptShortcutErr = ""
		return nil
	}
	if err := h.app.GlobalShortcut.Register(next, func() {
		application.InvokeSync(h.togglePrompt)
	}); err != nil {
		h.promptShortcutErr = err.Error()
		return err
	}
	if old := h.promptRegistered; old != "" {
		if err := h.app.GlobalShortcut.Unregister(old); err != nil {
			_ = h.app.GlobalShortcut.Unregister(next)
			h.promptShortcutErr = err.Error()
			return err
		}
	}
	h.promptRegistered = next
	h.promptShortcutErr = ""
	return nil
}

// PromptShortcut says the shortcut in use, or why none is.
func (h *host) PromptShortcut() (string, string) {
	h.promptMu.Lock()
	defer h.promptMu.Unlock()
	return h.promptRegistered, h.promptShortcutErr
}

// PromptCanInsert says whether this platform can put text in the app that
// was in front, rather than only copying it.
func (h *host) PromptCanInsert() bool {
	h.promptMu.Lock()
	app := h.promptApp
	h.promptMu.Unlock()
	return app.ID != "" && promptPasteReady()
}

// PromptNeedsAccess says whether the system is withholding permission to
// send the paste key.
func (h *host) PromptNeedsAccess() bool { return promptNeedsAccess() }

// ShowPrompt opens the picker. The shortcut toggles it instead.
func (h *host) ShowPrompt() { application.InvokeSync(func() { h.showPromptNow(false) }) }

// HidePrompt closes the picker and gives the front application its focus.
func (h *host) HidePrompt() {
	application.InvokeSync(func() {
		app := h.promptApp
		if h.prompt != nil {
			h.prompt.Hide()
		}
		if app.ID != "" {
			activatePromptApp(app)
		}
	})
}

func (h *host) togglePrompt() {
	if h.prompt == nil {
		return
	}
	h.showPromptNow(windowOpen(h.prompt))
}

func (h *host) showPromptNow(hide bool) {
	if h.prompt == nil {
		return
	}
	if hide {
		app := h.promptApp
		h.prompt.Hide()
		if app.ID != "" {
			activatePromptApp(app)
		}
		return
	}
	h.promptMu.Lock()
	h.promptApp = capturePromptApp()
	h.promptMu.Unlock()
	h.revealPrompt()
}

// revealPrompt shows the window for the application captured when it opened.
func (h *host) revealPrompt() {
	screen, err := h.prompt.GetScreen()
	if err != nil || screen == nil {
		screen = h.app.Screen.GetPrimary()
	}
	if screen != nil {
		a := screen.WorkArea
		h.prompt.SetScreen(screen)
		h.prompt.Center()
		x, _ := h.prompt.Position()
		y := a.Y + a.Height/7
		h.prompt.SetPosition(x, y)
	}
	h.prompt.Show().Focus()
}

// InsertPrompt puts text into the app that was in front before the picker
// took focus. If automatic insertion is not possible, it copies the text and
// leaves the picker open so the page can say what to do next.
func (h *host) InsertPrompt(text string) promptInsertResult {
	h.promptMu.Lock()
	app := h.promptApp
	h.promptMu.Unlock()
	if app.ID == "" || !promptPasteReady() {
		if !h.Copy(text) {
			return promptInsertResult{Error: "The prompt could not be put on the clipboard."}
		}
		return promptInsertResult{
			Copied:      true,
			NeedsAccess: promptNeedsAccess(),
			Error:       "Automatic insertion is not available. The prompt is on the clipboard.",
		}
	}

	old, hadText := h.app.Clipboard.Text()
	if !h.Copy(text) {
		return promptInsertResult{Error: "The prompt could not be put on the clipboard."}
	}
	application.InvokeSync(func() {
		if h.prompt != nil {
			h.prompt.Hide()
		}
	})
	time.Sleep(80 * time.Millisecond)
	application.InvokeSync(func() { activatePromptApp(app) })
	time.Sleep(90 * time.Millisecond)
	var pasteErr error
	application.InvokeSync(func() { pasteErr = pasteIntoPromptApp(app) })
	if pasteErr != nil {
		application.InvokeSync(h.revealPrompt)
		return promptInsertResult{Copied: true, Error: "Couldn't insert the prompt. It is on the clipboard."}
	}
	time.Sleep(220 * time.Millisecond)
	if hadText {
		h.app.Clipboard.SetText(old)
	}
	h.promptMu.Lock()
	h.promptApp = promptApp{}
	h.promptMu.Unlock()
	return promptInsertResult{Inserted: true}
}

// OpenPromptAccessibilitySettings brings the system page that grants
// magpie permission to send a paste key.
func (h *host) OpenPromptAccessibilitySettings() { openPromptAccessibilitySettings() }
