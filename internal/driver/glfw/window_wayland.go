//go:build wayland && (linux || freebsd || openbsd || netbsd)

package glfw

import (
	"fyne.io/fyne/v2/driver"
)

// GetWindowHandle returns the window handle. Only implemented for X11 currently.
func (w *window) GetWindowHandle() string {
	return "" // TODO: Find a way to get the Wayland handle for xdg_foreign protocol. Return "wayland:{id}".
}

// assert we are implementing driver.NativeWindow
var _ driver.NativeWindow = (*window)(nil)

func (w *window) RunNative(f func(any) error) error {
	var err error
	done := make(chan struct{})
	runOnMain(func() {
		err = f(driver.WaylandWindowContext{
			WaylandSurface: uintptr(w.view().GetWaylandWindow()),
			EGLSurface:     uintptr(w.view().GetEGLSurface()),
		})
		close(done)
	})
	<-done
	return err
}
