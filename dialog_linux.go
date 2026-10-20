package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdlib.h>

static GSList *runOpenDialog(const char *dir) {
	GtkWidget *dlg = gtk_file_chooser_dialog_new("Open File", NULL,
		GTK_FILE_CHOOSER_ACTION_OPEN,
		"_Cancel", GTK_RESPONSE_CANCEL,
		"_Open", GTK_RESPONSE_ACCEPT,
		NULL);
	gtk_file_chooser_set_select_multiple(GTK_FILE_CHOOSER(dlg), TRUE);
	if (dir != NULL) gtk_file_chooser_set_current_folder(GTK_FILE_CHOOSER(dlg), dir);
	GSList *files = NULL;
	if (gtk_dialog_run(GTK_DIALOG(dlg)) == GTK_RESPONSE_ACCEPT) {
		files = gtk_file_chooser_get_filenames(GTK_FILE_CHOOSER(dlg));
	}
	gtk_widget_destroy(dlg);
	return files;
}

static void freeFilenames(GSList *files) {
	g_slist_free_full(files, g_free);
}
*/
import "C"

import "unsafe"

// pickFiles shows the native open dialog starting in dir. Must run on the main thread.
func pickFiles(dir string) []string {
	var cdir *C.char
	if dir != "" {
		cdir = C.CString(dir)
		defer C.free(unsafe.Pointer(cdir))
	}
	files := C.runOpenDialog(cdir)
	defer C.freeFilenames(files)
	var paths []string
	for l := files; l != nil; l = l.next {
		paths = append(paths, C.GoString((*C.char)(l.data)))
	}
	return paths
}
