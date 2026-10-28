# Prompts and the prompt picker

Prompts are reusable text the user saves for themselves. They live under
Library → Prompts, but are not written into agents and do not take part in
the Library's agent sync.

## Responsibilities and sources of truth

| Part | Responsibility | Source |
| --- | --- | --- |
| prompts.json | `Prompt` holds an id, name and text. The file is beside settings and is written atomically with mode 0600. A missing file is an empty list; a broken file is an error and is never overwritten. | [`prompts.go`](../../internal/prompts/prompts.go) |
| API | `GET /api/prompts` lists prompts and the picker state. `save`, `remove`, `insert`, `show`, `hide` and `accessibility` are the picker's actions. | [`prompts.go`](../../internal/gui/prompts.go) |
| Library page | The Prompts tab lists, adds, edits and removes prompts, and shows the global shortcut. Settings changes the shortcut. | [`client.js`](../../internal/gui/assets/plugins/library/client.js), [`app.js`](../../internal/gui/assets/app.js) |
| Picker window | A hidden, frameless, always-on-top window loads `prompt.html`. The shortcut captures the application in front, shows the window centered near the top of the screen, and leaves the search input focused. | [`promptwindow.go`](../../internal/gui/promptwindow.go), [`prompt.html`](../../internal/gui/assets/prompt.html), [`prompt.js`](../../internal/gui/assets/prompt.js) |
| Shortcut | The default is `CmdOrCtrl+Shift+Space`, kept in `settings.PromptShortcut`. A new shortcut is registered before the old one is released, so an invalid or already-owned key leaves the working one in place. | [`settings.go`](../../internal/settings/settings.go), [`promptwindow.go`](../../internal/gui/promptwindow.go) |
| Insertion | The prompt goes on the clipboard, the picker hides, the captured application is activated, and a paste key is sent. A text clipboard value present before is put back after the paste. | [`promptwindow.go`](../../internal/gui/promptwindow.go), platform files beside it |

## Runtime paths

1. **Opening.** The global shortcut runs `togglePrompt`; it captures the
   front application before showing the picker. A picker already open is
   hidden and the captured application is activated again.
2. **Filtering.** The page reads `/api/prompts`, matches every word against
   the prompt's name and text, and ranks a name match above a body match.
   Arrow keys move the selection, Enter inserts, Escape hides.
3. **Inserting.** `/api/prompts/insert` reads the saved prompt by id. macOS
   checks Accessibility permission, uses CoreGraphics to send Command-V,
   and restores a prior text clipboard value. Windows activates the prior
   window and sends Ctrl-V. A platform without a supported paste path
   copies the prompt and returns `needsAccess` or a fallback error instead
   of claiming insertion worked.
4. **Changing the shortcut.** Settings registers the replacement first,
   then unregisters the old binding and saves the setting. A failed
   registration is shown in the row and the previous binding remains.

## Constraints and failure behavior

- Prompts are local text. They are not sent to a model by magpie, and the
  feature does not add telemetry.
- `prompts.json` is separate from `library.json`: editing a prompt must not
  run the Library sync that writes agents' files.
- A malformed `prompts.json` returns an error from reads and writes. Magpie
  does not replace it with an empty list.
- macOS automatic insertion needs Accessibility permission. Without it, the
  picker copies the prompt and offers System Settings instead.
- A non-text clipboard item cannot be restored through the text clipboard
  API. The selected prompt remains on the clipboard in that case.
- Linux currently uses the copy-and-paste fallback; synthetic input is not
  assumed to be available across Wayland compositors.

## Verification

```sh
go test ./internal/prompts
go test ./internal/gui -run 'TestPrompts|TestPromptShortcutSetting'
NODE_PATH="$(npm root -g)" node --test internal/gui/tests/prompts.test.cjs
```

The browser test covers both engines, all five GUI languages, keyboard
selection, the empty and fallback states, and the Library editor.
