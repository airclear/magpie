//go:build (!darwin && !windows) || nogui

package gui

func capturePromptApp() promptApp { return promptApp{} }

func activatePromptApp(promptApp) {}

func pasteIntoPromptApp(promptApp) error { return errPromptInsert }

func promptPasteReady() bool { return false }

func promptNeedsAccess() bool { return false }

func openPromptAccessibilitySettings() {}
