//go:build !windows

package main

import _ "embed"
import "github.com/getlantern/systray"

//go:embed assets/tray.png
var trayIcon []byte

func setPlatformTrayIcon() {
	if len(trayIcon) > 0 {
		systray.SetIcon(trayIcon)
	}
}
