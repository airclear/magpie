//go:build darwin && cgo && !nogui

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices
#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>

static int promptFrontPID(void) {
	@autoreleasepool {
		NSRunningApplication *app = NSWorkspace.sharedWorkspace.frontmostApplication;
		return app == nil ? 0 : app.processIdentifier;
	}
}

static void promptFrontName(char *dst, int n) {
	@autoreleasepool {
		NSRunningApplication *app = NSWorkspace.sharedWorkspace.frontmostApplication;
		NSString *name = app == nil ? @"" : app.localizedName;
		snprintf(dst, n, "%s", name.UTF8String ?: "");
	}
}

static int promptTrusted(bool ask) {
	@autoreleasepool {
		NSDictionary *opts = @{(__bridge id)kAXTrustedCheckOptionPrompt: @(ask)};
		return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts) ? 1 : 0;
	}
}

static void promptActivate(int pid) {
	@autoreleasepool {
		NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
		[app activateWithOptions:NSApplicationActivateAllWindows];
	}
}

static void promptPaste(void) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, 9, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, 9, false);
	if (down != NULL) {
		CGEventSetFlags(down, kCGEventFlagMaskCommand);
		CGEventPost(kCGHIDEventTap, down);
		CFRelease(down);
	}
	if (up != NULL) {
		CGEventSetFlags(up, kCGEventFlagMaskCommand);
		CGEventPost(kCGHIDEventTap, up);
		CFRelease(up);
	}
}

static void promptOpenAccessibility(void) {
	@autoreleasepool {
		NSURL *url = [NSURL URLWithString:@"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"];
		[NSWorkspace.sharedWorkspace openURL:url];
	}
}
*/
import "C"

import (
	"fmt"
	"strconv"
)

func capturePromptApp() promptApp {
	pid := int(C.promptFrontPID())
	if pid == 0 {
		return promptApp{}
	}
	var name [256]C.char
	C.promptFrontName(&name[0], C.int(len(name)))
	return promptApp{ID: strconv.Itoa(pid), Name: C.GoString(&name[0])}
}

func activatePromptApp(app promptApp) {
	pid, err := strconv.Atoi(app.ID)
	if err == nil && pid > 0 {
		C.promptActivate(C.int(pid))
	}
}

func pasteIntoPromptApp(promptApp) error {
	if C.promptTrusted(false) == 0 {
		return fmt.Errorf("magpie needs Accessibility access to paste into another application")
	}
	C.promptPaste()
	return nil
}

func promptPasteReady() bool { return C.promptTrusted(false) != 0 }

func promptNeedsAccess() bool { return !promptPasteReady() }

func openPromptAccessibilitySettings() {
	_ = C.promptTrusted(true)
	C.promptOpenAccessibility()
}
