//go:build windows && !nogui

package gui

import (
	"strconv"
	"syscall"
	"unsafe"
)

var (
	promptUser32         = syscall.NewLazyDLL("user32.dll")
	promptGetForeground  = promptUser32.NewProc("GetForegroundWindow")
	promptSetForeground  = promptUser32.NewProc("SetForegroundWindow")
	promptShowWindow     = promptUser32.NewProc("ShowWindow")
	promptGetWindowTextW = promptUser32.NewProc("GetWindowTextW")
	promptKeybdEvent     = promptUser32.NewProc("keybd_event")
)

const (
	promptSWRestore  = 9
	promptVKControl  = 0x11
	promptVKV        = 0x56
	promptKeyeventUp = 0x0002
)

func capturePromptApp() promptApp {
	hwnd, _, _ := promptGetForeground.Call()
	if hwnd == 0 {
		return promptApp{}
	}
	var buf [256]uint16
	n, _, _ := promptGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return promptApp{ID: strconv.FormatUint(uint64(hwnd), 10), Name: syscall.UTF16ToString(buf[:n])}
}

func activatePromptApp(app promptApp) {
	hwnd, err := strconv.ParseUint(app.ID, 10, 64)
	if err == nil && hwnd != 0 {
		promptShowWindow.Call(uintptr(hwnd), promptSWRestore)
		promptSetForeground.Call(uintptr(hwnd))
	}
}

func pasteIntoPromptApp(promptApp) error {
	promptKeybdEvent.Call(promptVKControl, 0, 0, 0)
	promptKeybdEvent.Call(promptVKV, 0, 0, 0)
	promptKeybdEvent.Call(promptVKV, 0, promptKeyeventUp, 0)
	promptKeybdEvent.Call(promptVKControl, 0, promptKeyeventUp, 0)
	return nil
}

func promptPasteReady() bool { return true }

func promptNeedsAccess() bool { return false }

func openPromptAccessibilitySettings() {}
