//go:build !ci && !js && !wasm && !test_web_driver && (linux || openbsd || freebsd || netbsd) && !android
// +build !ci
// +build !js
// +build !wasm
// +build !test_web_driver
// +build linux openbsd freebsd netbsd
// +build !android

package app

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/execabs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var once sync.Once

func defaultVariant() fyne.ThemeVariant {
	return findFreedestktopColorScheme()
}

func (a *fyneApp) OpenURL(url *url.URL) error {
	if openURLThroughPortal(url) == nil {
		return nil
	}

	cmd := execabs.Command("xdg-open", url.String())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Start()
}

func openURLThroughPortal(url *url.URL) error {
	conn, err := dbus.SessionBus() // shared connection, don't close
	if err != nil {
		fyne.LogError("Unable to connect to session D-Bus", err)
		return err
	}

	parentWindow := ""
	uri := url.String()
	options := map[string]dbus.Variant{}

	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	call := obj.Call("org.freedesktop.portal.OpenURI.OpenURI", 0, parentWindow, uri, options)
	if call.Err != nil {
		fyne.LogError("Failed to open url with xdg-desktop-portal", call.Err)
		return call.Err
	}

	return nil
}

// fetch color variant from dbus portal desktop settings.
func findFreedestktopColorScheme() fyne.ThemeVariant {
	dbusConn, err := dbus.SessionBus()
	if err != nil {
		fyne.LogError("Unable to connect to session D-Bus", err)
		return theme.VariantDark
	}

	dbusObj := dbusConn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	call := dbusObj.Call(
		"org.freedesktop.portal.Settings.ReadOne",
		dbus.FlagNoAutoStart,
		"org.freedesktop.appearance",
		"color-scheme",
	)
	if call.Err != nil {
		// many desktops don't have this exported yet
		return theme.VariantDark
	}

	var value uint8
	if err = call.Store(&value); err != nil {
		fyne.LogError("failed to read theme variant from D-Bus", err)
		return theme.VariantDark
	}

	// See: https://github.com/flatpak/xdg-desktop-portal/blob/1.16.0/data/org.freedesktop.impl.portal.Settings.xml#L32-L46
	// 0: No preference
	// 1: Prefer dark appearance
	// 2: Prefer light appearance
	switch value {
	case 2:
		return theme.VariantLight
	case 1:
		return theme.VariantDark
	default:
		// Default to light theme to support Gnome's default see https://github.com/fyne-io/fyne/pull/3561
		return theme.VariantLight
	}
}

func (a *fyneApp) SendNotification(n *fyne.Notification) {
	conn, err := dbus.SessionBus() // shared connection, don't close
	if err != nil {
		fyne.LogError("Unable to connect to session D-Bus", err)
		return
	}

	if a.sendNotificationThroughPortal(conn, n) == nil {
		return // No need to use the fallback path.
	}

	appIcon := a.cachedIconPath()
	timeout := int32(0) // we don't support this yet

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0, a.uniqueID, uint32(0),
		appIcon, n.Title, n.Content, []string{}, map[string]dbus.Variant{}, timeout)
	if call.Err != nil {
		fyne.LogError("Failed to send message to bus", call.Err)
	}
}

// Sending with same ID replaces the old notification.
var notificationID uint64 = 0

// See https://flatpak.github.io/xdg-desktop-portal/docs/#gdbus-org.freedesktop.portal.Notification.
func (a *fyneApp) sendNotificationThroughPortal(conn *dbus.Conn, n *fyne.Notification) error {
	id := strconv.FormatUint(notificationID, 10)
	data := map[string]dbus.Variant{
		"title": dbus.MakeVariant(n.Title),
		"body":  dbus.MakeVariant(n.Content),
		"icon":  dbus.MakeVariant(a.uniqueID),
	}

	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	call := obj.Call("org.freedesktop.portal.Notification.AddNotification", 0, id, data)
	if call.Err != nil {
		fyne.LogError("Failed to send notification to xdg-desktop-portal", call.Err)
		return call.Err
	}

	notificationID++
	return nil
}

func (a *fyneApp) saveIconToCache(dirPath, filePath string) error {
	err := os.MkdirAll(dirPath, 0700)
	if err != nil {
		fyne.LogError("Unable to create application cache directory", err)
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		fyne.LogError("Unable to create icon file", err)
		return err
	}

	defer file.Close()

	if icon := a.Icon(); icon != nil {
		_, err = file.Write(icon.Content())
		if err != nil {
			fyne.LogError("Unable to write icon contents", err)
			return err
		}
	}

	return nil
}

func (a *fyneApp) cachedIconPath() string {
	if a.Icon() == nil {
		return ""
	}

	dirPath := filepath.Join(rootCacheDir(), a.UniqueID())
	filePath := filepath.Join(dirPath, "icon.png")
	once.Do(func() {
		err := a.saveIconToCache(dirPath, filePath)
		if err != nil {
			filePath = ""
		}
	})

	return filePath
}

// SetSystemTrayMenu creates a system tray item and attaches the specified menu.
// By default this will use the application icon.
func (a *fyneApp) SetSystemTrayMenu(menu *fyne.Menu) {
	if desk, ok := a.Driver().(systrayDriver); ok { // don't use this on mobile tag
		desk.SetSystemTrayMenu(menu)
	}
}

// SetSystemTrayIcon sets a custom image for the system tray icon.
// You should have previously called `SetSystemTrayMenu` to initialise the menu icon.
func (a *fyneApp) SetSystemTrayIcon(icon fyne.Resource) {
	if desk, ok := a.Driver().(systrayDriver); ok { // don't use this on mobile tag
		desk.SetSystemTrayIcon(icon)
	}
}

func rootConfigDir() string {
	desktopConfig, _ := os.UserConfigDir()
	return filepath.Join(desktopConfig, "fyne")
}

func rootCacheDir() string {
	desktopCache, _ := os.UserCacheDir()
	return filepath.Join(desktopCache, "fyne")
}

func watchTheme() {
	go watchFreedekstopThemeChange()
}

func themeChanged() {
	fyne.CurrentApp().Settings().(*settings).setupTheme()
}

// connect to dbus to detect color-schem theme changes in portal settings.
func watchFreedekstopThemeChange() {
	conn, err := dbus.SessionBus()
	if err != nil {
		fyne.LogError("Unable to connect to session D-Bus", err)
		return
	}

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/portal/desktop"),
		dbus.WithMatchInterface("org.freedesktop.portal.Settings"),
		dbus.WithMatchMember("SettingChanged"),
	); err != nil {
		fyne.LogError("D-Bus signal match failed", err)
		return
	}
	defer conn.Close()

	dbusChan := make(chan *dbus.Signal)
	conn.Signal(dbusChan)

	for sig := range dbusChan {
		for _, v := range sig.Body {
			if v == "color-scheme" {
				themeChanged()
				break
			}
		}
	}
}
