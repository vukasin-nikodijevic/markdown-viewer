package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// Cocoa delivers Cmd+C/V/X/A/Z through the main menu's key equivalents;
// webview creates no menu, so without this the standard edit shortcuts do nothing.
static void setupMainMenu(void) {
	NSMenu *bar = [[NSMenu alloc] init];

	NSMenuItem *appItem = [[NSMenuItem alloc] init];
	NSMenu *appMenu = [[NSMenu alloc] init];
	[appMenu addItemWithTitle:@"Hide md-viewer" action:@selector(hide:) keyEquivalent:@"h"];
	[appMenu addItem:[NSMenuItem separatorItem]];
	[appMenu addItemWithTitle:@"Quit md-viewer" action:@selector(terminate:) keyEquivalent:@"q"];
	[appItem setSubmenu:appMenu];
	[bar addItem:appItem];

	NSMenuItem *editItem = [[NSMenuItem alloc] init];
	NSMenu *editMenu = [[NSMenu alloc] initWithTitle:@"Edit"];
	[editMenu addItemWithTitle:@"Undo" action:@selector(undo:) keyEquivalent:@"z"];
	[editMenu addItemWithTitle:@"Redo" action:@selector(redo:) keyEquivalent:@"Z"];
	[editMenu addItem:[NSMenuItem separatorItem]];
	[editMenu addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
	[editMenu addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
	[editMenu addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
	[editMenu addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];
	[editItem setSubmenu:editMenu];
	[bar addItem:editItem];

	[NSApp setMainMenu:bar];
}
*/
import "C"

// setupMenu installs the macOS menu bar. Must run on the main thread after webview.New.
func setupMenu() {
	C.setupMainMenu()
}
