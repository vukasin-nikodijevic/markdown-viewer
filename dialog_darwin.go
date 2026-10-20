package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// Returns the chosen paths joined by '\n' (caller frees), or NULL if cancelled.
static char *runOpenPanel(const char *dir) {
	@autoreleasepool {
		NSOpenPanel *panel = [NSOpenPanel openPanel];
		panel.canChooseFiles = YES;
		panel.canChooseDirectories = NO;
		panel.allowsMultipleSelection = YES;
		if (dir != NULL) {
			panel.directoryURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:dir] isDirectory:YES];
		}
		if ([panel runModal] != NSModalResponseOK) return NULL;
		NSMutableArray *paths = [NSMutableArray array];
		for (NSURL *url in panel.URLs) [paths addObject:url.path];
		return strdup([[paths componentsJoinedByString:@"\n"] UTF8String]);
	}
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

// pickFiles shows the native open dialog starting in dir. Must run on the main thread.
func pickFiles(dir string) []string {
	var cdir *C.char
	if dir != "" {
		cdir = C.CString(dir)
		defer C.free(unsafe.Pointer(cdir))
	}
	res := C.runOpenPanel(cdir)
	if res == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(res))
	return strings.Split(C.GoString(res), "\n")
}
