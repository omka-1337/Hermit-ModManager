//go:build linux && cgo && !gtk3

package platform

/*
#cgo pkg-config: gtk4
#include <stdlib.h>
#include <gtk/gtk.h>

// GTK before 4.16 tells KWin to draw a server-side frame around any window
// without a GTK titlebar, including undecorated ones, whenever the window is
// realized. A hidden, empty titlebar makes GTK announce client-side
// decorations instead; undecorated windows get no CSD shadow, so nothing is
// drawn.
static void hermit_prepare_frameless(void *window, const char *title) {
	if (window == NULL)
		return;
	gtk_window_set_title(GTK_WINDOW(window), title);
	if (gtk_get_major_version() != 4 || gtk_get_minor_version() >= 16)
		return;
	GtkWidget *titlebar = gtk_box_new(GTK_ORIENTATION_HORIZONTAL, 0);
	gtk_widget_set_visible(titlebar, FALSE);
	gtk_window_set_titlebar(GTK_WINDOW(window), titlebar);
}
*/
import "C"

import "unsafe"

// PrepareFramelessWindow sets the window title and works around old GTK
// versions (the AppImage bundles GTK 4.14) that give frameless windows a KDE
// window frame. It must run on the GTK thread before the window is shown.
func PrepareFramelessWindow(nativeWindow unsafe.Pointer, title string) {
	t := C.CString(title)
	defer C.free(unsafe.Pointer(t))
	C.hermit_prepare_frameless(nativeWindow, t)
}

// SetProgramName names the process for the desktop (taskbar, window class);
// inside the AppImage it would otherwise be "AppRun.wrapped".
func SetProgramName(name, displayName string) {
	n, d := C.CString(name), C.CString(displayName)
	defer C.free(unsafe.Pointer(n))
	defer C.free(unsafe.Pointer(d))
	C.g_set_prgname(n)
	C.g_set_application_name(d)
}
